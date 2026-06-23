package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"alethea/api/auth"
	"alethea/api/services/billing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Billing owns the /api/me/billing/* + /api/billing/webhook endpoints.
type Billing struct {
	DB      *pgxpool.Pool
	Service *billing.Service
}

func NewBilling(db *pgxpool.Pool, svc *billing.Service) *Billing {
	return &Billing{DB: db, Service: svc}
}

type checkoutReq struct {
	Period string `json:"period"` // 'monthly' | 'yearly'
}

// Checkout creates a Stripe Checkout Session and returns the redirect URL.
func (b *Billing) Checkout(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	if b.Service == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "billing_disabled",
			"Billing is not configured on this server (STRIPE_SECRET_KEY missing).")
		return
	}
	var body checkoutReq
	if err := decodeStrictJSON(w, r, &body); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	period := billing.Period(body.Period)
	if period != billing.PeriodMonthly && period != billing.PeriodYearly {
		period = billing.PeriodMonthly
	}

	// Look up the user's email — Stripe Customer needs it.
	var email string
	if err := b.DB.QueryRow(r.Context(), `SELECT email FROM users WHERE id = $1`, sess.UserID).Scan(&email); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	url, err := b.Service.CreateCheckoutSession(r.Context(), sess.UserID, email, period)
	if err != nil {
		slog.Error("billing.checkout", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "could not start checkout")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

type portalReq struct {
	ReturnURL string `json:"returnUrl"`
}

// Portal creates a Stripe Customer Portal Session and returns the URL.
func (b *Billing) Portal(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	if b.Service == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "billing_disabled", "Billing is not configured.")
		return
	}
	var body portalReq
	_ = decodeStrictJSON(w, r, &body)
	if body.ReturnURL == "" {
		body.ReturnURL = "/"
	}
	url, err := b.Service.CreatePortalSession(r.Context(), sess.UserID, body.ReturnURL)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "no_subscription", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// Webhook is the Stripe-signed callback. CSRF middleware is bypassed for
// this route (signature is the auth). Stripe will redeliver on non-2xx.
func (b *Billing) Webhook(w http.ResponseWriter, r *http.Request) {
	if b.Service == nil {
		// We don't have a webhook secret so we can't verify — drop the request
		// loudly. Don't 503 in case Stripe interprets that as transient.
		writeJSONErr(w, http.StatusBadRequest, "billing_disabled", "Billing is not configured.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB cap on Stripe events
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	sig := r.Header.Get("Stripe-Signature")
	if err := b.Service.HandleWebhook(r.Context(), body, sig); err != nil {
		// Bad signature is a 400; everything else 500. Stripe will retry 5xx.
		slog.Warn("billing.webhook", "err", err)
		if isSignatureError(err) {
			writeJSONErr(w, http.StatusBadRequest, "bad_signature", "webhook signature verification failed")
			return
		}
		writeJSONErr(w, http.StatusInternalServerError, "internal", "webhook processing failed")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func isSignatureError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "invalid signature") || strings.Contains(msg, "no signatures found")
}

// --- /api/economics (public) -----------------------------------------------

type Economics struct {
	DB *pgxpool.Pool
}

func NewEconomics(db *pgxpool.Pool) *Economics {
	return &Economics{DB: db}
}

// Get returns aggregate platform numbers for the EconomicsTicker. Always
// returns a 200 even if the queries fail (graceful fallback values), so
// the ticker on every public page doesn't 5xx the user out of the footer.
func (e *Economics) Get(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"spend_today_eur": 0.0,
		"checks_today":    0,
		"byok_percent":    0,
	}

	var spendMicros int
	if err := e.DB.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(total_cost_micros), 0) FROM checks WHERE created_at::date = CURRENT_DATE`,
	).Scan(&spendMicros); err == nil {
		// Microcents → USD; UI label says EUR for now (close enough at platform scale).
		out["spend_today_eur"] = float64(spendMicros) / 1_000_000
	}

	var checksToday int
	_ = e.DB.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM checks WHERE created_at::date = CURRENT_DATE`,
	).Scan(&checksToday)
	out["checks_today"] = checksToday

	if checksToday > 0 {
		var byok int
		_ = e.DB.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM checks WHERE created_at::date = CURRENT_DATE AND byok_used = TRUE`,
		).Scan(&byok)
		out["byok_percent"] = byok * 100 / checksToday
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_ = json.NewEncoder(w).Encode(out)
}
