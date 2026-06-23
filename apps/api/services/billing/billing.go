// Package billing wraps Stripe for the Plus subscription tier.
//
//   - Service struct = config + Stripe API key, no global state
//   - CreateCheckoutSession launches the user into Stripe-hosted Checkout
//   - CreatePortalSession launches them into Stripe-hosted Customer Portal
//   - HandleWebhook validates the signature, idempotently processes the
//     event (via plan_events.stripe_event_id UNIQUE constraint), updates
//     users.plan, writes an audit log row.
//
// No raw card data ever touches Alethea. We use Stripe Checkout + Portal
// to keep the PCI scope at SAQ-A. Webhook signature verification is the
// only auth on the webhook endpoint (CSRF middleware is bypassed for it
// by route registration in main.go).
package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"alethea/api/audit"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	stripe "github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
	portalsession "github.com/stripe/stripe-go/v79/billingportal/session"
	"github.com/stripe/stripe-go/v79/webhook"
)

// Config is read from env at app startup. STRIPE_SECRET_KEY is required;
// missing keys make the Service nil and /api/me/billing/* return 503.
type Config struct {
	SecretKey      string // STRIPE_SECRET_KEY
	WebhookSecret  string // STRIPE_WEBHOOK_SECRET
	PriceMonthly   string // STRIPE_PRICE_PLUS_MONTHLY
	PriceYearly    string // STRIPE_PRICE_PLUS_YEARLY
	SuccessURL     string // where Stripe redirects after a successful checkout
	CancelURL      string // where Stripe redirects on user cancel
}

func (c Config) Enabled() bool {
	return c.SecretKey != "" && c.WebhookSecret != "" && c.PriceMonthly != ""
}

// Service is the Stripe wrapper. Hold one per process.
type Service struct {
	cfg  Config
	pool *pgxpool.Pool
}

func NewService(cfg Config, pool *pgxpool.Pool) *Service {
	if !cfg.Enabled() {
		return nil
	}
	stripe.Key = cfg.SecretKey
	return &Service{cfg: cfg, pool: pool}
}

// Period selects the price. Stripe expects an exact price ID.
type Period string

const (
	PeriodMonthly Period = "monthly"
	PeriodYearly  Period = "yearly"
)

func (s *Service) priceID(p Period) string {
	if p == PeriodYearly && s.cfg.PriceYearly != "" {
		return s.cfg.PriceYearly
	}
	return s.cfg.PriceMonthly
}

// CreateCheckoutSession returns the URL Stripe wants the user redirected to.
// We pass user_id in metadata so the webhook can resolve back to our row
// without trusting client state.
func (s *Service) CreateCheckoutSession(ctx context.Context, userID int64, userEmail string, period Period) (string, error) {
	if s == nil {
		return "", errors.New("billing disabled")
	}
	priceID := s.priceID(period)
	if priceID == "" {
		return "", errors.New("no Stripe price configured for the selected period")
	}
	customerID, err := s.ensureCustomer(ctx, userID, userEmail)
	if err != nil {
		return "", err
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer: stripe.String(customerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		SuccessURL: stripe.String(s.cfg.SuccessURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.cfg.CancelURL),
		Metadata: map[string]string{
			"alethea_user_id": fmt.Sprintf("%d", userID),
			"alethea_period":  string(period),
		},
	}
	params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
		Metadata: map[string]string{
			"alethea_user_id": fmt.Sprintf("%d", userID),
		},
	}
	params.Context = ctx

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe checkout: %w", err)
	}
	return sess.URL, nil
}

// CreatePortalSession returns the URL for the Stripe-hosted customer portal
// (manage subscription / change payment method / cancel).
func (s *Service) CreatePortalSession(ctx context.Context, userID int64, returnURL string) (string, error) {
	if s == nil {
		return "", errors.New("billing disabled")
	}
	customerID, err := s.lookupCustomer(ctx, userID)
	if err != nil {
		return "", err
	}
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}
	params.Context = ctx
	ps, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe portal: %w", err)
	}
	return ps.URL, nil
}

// HandleWebhook verifies the Stripe signature, idempotently processes the
// event, transitions the user's plan, and writes an audit log.
//
// Stripe will redeliver events freely on non-2xx responses, so the
// stripe_event_id UNIQUE constraint on plan_events is the idempotency
// guarantee: a redelivered event INSERTs once, then ON CONFLICT noops.
func (s *Service) HandleWebhook(ctx context.Context, rawBody []byte, signature string) error {
	if s == nil {
		return errors.New("billing disabled")
	}
	event, err := webhook.ConstructEvent(rawBody, signature, s.cfg.WebhookSecret)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	// Idempotency: try to insert the event ID first; on conflict, this is a
	// redelivery — just return 200 success.
	payload, _ := json.Marshal(event.Data.Raw)
	var existing bool
	err = s.pool.QueryRow(ctx,
		`INSERT INTO plan_events (stripe_event_id, kind, payload_json)
		 VALUES ($1, $2, $3::jsonb)
		 ON CONFLICT (stripe_event_id) DO NOTHING
		 RETURNING false`,
		event.ID, event.Type, payload,
	).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already-processed event. Return success so Stripe stops redelivering.
		slog.Info("stripe webhook: duplicate event ignored", "id", event.ID, "type", event.Type)
		return nil
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// belt-and-suspenders: another race-conditioned redelivery
			return nil
		}
		return fmt.Errorf("plan_events insert: %w", err)
	}

	// Dispatch.
	switch event.Type {
	case "checkout.session.completed":
		return s.onCheckoutCompleted(ctx, event)
	case "customer.subscription.updated":
		return s.onSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		return s.onSubscriptionDeleted(ctx, event)
	case "invoice.payment_failed":
		return s.onPaymentFailed(ctx, event)
	default:
		// Quietly accept everything else (Stripe sends lots we don't care about).
		return nil
	}
}

// --- Event handlers ---

func (s *Service) onCheckoutCompleted(ctx context.Context, ev stripe.Event) error {
	userID, ok := userIDFromMetadata(ev)
	if !ok {
		slog.Warn("stripe webhook: checkout.session.completed without user_id metadata", "id", ev.ID)
		return nil
	}
	return s.setPlan(ctx, userID, "plus", "stripe.checkout.completed", ev.ID)
}

func (s *Service) onSubscriptionUpdated(ctx context.Context, ev stripe.Event) error {
	// If status is active or trialing → plus; else → free.
	var sub struct {
		Status   string `json:"status"`
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal(ev.Data.Raw, &sub); err != nil {
		return err
	}
	userID, err := s.lookupUserByCustomer(ctx, sub.Customer)
	if err != nil {
		slog.Warn("stripe webhook: customer.subscription.updated unknown customer", "id", ev.ID, "customer", sub.Customer)
		return nil
	}
	plan := "free"
	if sub.Status == "active" || sub.Status == "trialing" {
		plan = "plus"
	}
	return s.setPlan(ctx, userID, plan, "stripe.subscription.updated:"+sub.Status, ev.ID)
}

func (s *Service) onSubscriptionDeleted(ctx context.Context, ev stripe.Event) error {
	var sub struct {
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal(ev.Data.Raw, &sub); err != nil {
		return err
	}
	userID, err := s.lookupUserByCustomer(ctx, sub.Customer)
	if err != nil {
		return nil
	}
	return s.setPlan(ctx, userID, "free", "stripe.subscription.deleted", ev.ID)
}

func (s *Service) onPaymentFailed(ctx context.Context, ev stripe.Event) error {
	// Log + audit only; Stripe's dunning will retry. We don't immediately
	// downgrade because that would punish a momentary card decline.
	var inv struct {
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal(ev.Data.Raw, &inv); err != nil {
		return err
	}
	userID, _ := s.lookupUserByCustomer(ctx, inv.Customer)
	uid := &userID
	if userID == 0 {
		uid = nil
	}
	audit.Log(ctx, s.pool, uid, "billing.payment_failed", "user", fmt.Sprintf("%d", userID), nil, map[string]any{
		"stripe_event_id": ev.ID,
	})
	return nil
}

func (s *Service) setPlan(ctx context.Context, userID int64, plan, reason, eventID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET plan = $1, updated_at = NOW() WHERE id = $2 AND plan != $1`,
		plan, userID,
	)
	if err != nil {
		return fmt.Errorf("set plan: %w", err)
	}
	uid := userID
	audit.Log(ctx, s.pool, &uid, "billing.plan_change", "user", fmt.Sprintf("%d", userID), nil, map[string]any{
		"plan": plan, "reason": reason, "stripe_event_id": eventID,
	})
	return nil
}

// --- Customer lookups ---

func (s *Service) ensureCustomer(ctx context.Context, userID int64, email string) (string, error) {
	var existing string
	err := s.pool.QueryRow(ctx,
		`SELECT stripe_customer_id FROM stripe_customers WHERE user_id = $1`, userID,
	).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	// Create one.
	custParams := &stripe.CustomerParams{
		Email: stripe.String(email),
		Metadata: map[string]string{
			"alethea_user_id": fmt.Sprintf("%d", userID),
		},
	}
	custParams.Context = ctx
	cust, err := customer.New(custParams)
	if err != nil {
		return "", fmt.Errorf("stripe customer.New: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO stripe_customers (user_id, stripe_customer_id) VALUES ($1, $2)
		 ON CONFLICT (user_id) DO NOTHING`,
		userID, cust.ID,
	)
	if err != nil {
		// We've created a Stripe customer we can't remember — log loudly so
		// ops can dedupe. Stripe doesn't auto-clean these.
		slog.Error("stripe_customers insert failed after customer.New", "err", err, "stripe_customer", cust.ID)
		return cust.ID, err
	}
	return cust.ID, nil
}

func (s *Service) lookupCustomer(ctx context.Context, userID int64) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT stripe_customer_id FROM stripe_customers WHERE user_id = $1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("no Stripe customer for this user — checkout first")
	}
	return id, err
}

func (s *Service) lookupUserByCustomer(ctx context.Context, customerID string) (int64, error) {
	var uid int64
	err := s.pool.QueryRow(ctx, `SELECT user_id FROM stripe_customers WHERE stripe_customer_id = $1`, customerID).Scan(&uid)
	return uid, err
}

func userIDFromMetadata(ev stripe.Event) (int64, bool) {
	// Stripe events nest metadata in event.Data.Object. We look at the top level.
	var obj struct {
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(ev.Data.Raw, &obj); err != nil {
		return 0, false
	}
	raw, ok := obj.Metadata["alethea_user_id"]
	if !ok {
		return 0, false
	}
	var id int64
	if _, err := fmt.Sscanf(strings.TrimSpace(raw), "%d", &id); err != nil {
		return 0, false
	}
	return id, true
}
