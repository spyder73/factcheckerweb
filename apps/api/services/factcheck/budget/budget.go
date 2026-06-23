// Package budget enforces spend caps before launching a check. Addresses
// Phase A findings M4 (per-user daily $ budget + global circuit breaker)
// and M7 (per-check token + image caps — the API/handler layer enforces
// images, this package owns the dollar accounting).
//
// Sources of truth:
//   - per-user 24h spend: SUM(checks.total_cost_micros) WHERE user_id=$1 AND created_at > NOW()-24h
//   - global 24h spend:   SUM(checks.total_cost_micros) WHERE created_at > NOW()-24h
//
// No new tables, no Redis state, no cron — we already aggregate cost_micros
// into checks via totalCostFromDB at the end of every run. A SQL aggregate
// at the start of each check costs ~1ms and stays correct under restart.
package budget

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Caps in microcents (1 microcent = 1e-6 USD). 1_000_000 microcents = $1.
type Caps struct {
	PerPlanDailyMicros map[string]int // by plan: 'free','byok','plus','admin','anon'
	GlobalDailyMicros  int            // 0 = unlimited
	PerCheckMicros     int            // mid-pipeline circuit-break threshold (0 = no cap)
}

// DefaultCaps reads env overrides; sensible production-safe defaults otherwise.
// All values in MICROCENTS — convert from dollars at the env boundary.
func DefaultCaps() Caps {
	return Caps{
		PerPlanDailyMicros: map[string]int{
			"anon":  envDollarsToMicros("BUDGET_DAILY_USD_ANON", 0.50),    // $0.50/day
			"free":  envDollarsToMicros("BUDGET_DAILY_USD_FREE", 2.00),    // $2/day
			"byok":  envDollarsToMicros("BUDGET_DAILY_USD_BYOK", 0),       // 0 = unlimited (their wallet)
			"plus":  envDollarsToMicros("BUDGET_DAILY_USD_PLUS", 15.00),   // $15/day = ~$450/mo cap
			"admin": envDollarsToMicros("BUDGET_DAILY_USD_ADMIN", 0),      // unlimited
		},
		GlobalDailyMicros: envDollarsToMicros("BUDGET_GLOBAL_USD_DAILY", 200.00),  // $200/day platform-wide
		PerCheckMicros:    envDollarsToMicros("BUDGET_PER_CHECK_USD", 0.50),       // mid-check abort threshold
	}
}

// ErrUserOverBudget = the calling user has exhausted today's $ budget.
var ErrUserOverBudget = errors.New("budget: user over daily $ cap")

// ErrPlatformOverBudget = the platform-wide global cap is hit (cost circuit-break).
// Use this to return 503 — affects all users, not the calling one's fault.
var ErrPlatformOverBudget = errors.New("budget: platform over daily $ cap (circuit-breaker)")

// PreCheck verifies the user (if any) and the platform are under their daily
// dollar caps. Returns nil to proceed. Call BEFORE launching the pipeline.
// userID=nil means anonymous — global cap still checked.
func PreCheck(ctx context.Context, pool *pgxpool.Pool, caps Caps, userID *int64, plan string) error {
	// 1. Global platform cap
	if caps.GlobalDailyMicros > 0 {
		var globalSpend int
		err := pool.QueryRow(ctx,
			`SELECT COALESCE(SUM(total_cost_micros), 0)
			   FROM checks
			  WHERE created_at > NOW() - INTERVAL '24 hours'`,
		).Scan(&globalSpend)
		if err != nil {
			return fmt.Errorf("budget: query global: %w", err)
		}
		if globalSpend >= caps.GlobalDailyMicros {
			return ErrPlatformOverBudget
		}
	}

	// 2. Per-plan cap (anon = group by IP, but we just bypass with 0=unlimited
	//    and rely on rate-limiter for anon since we don't have user_id).
	cap, ok := caps.PerPlanDailyMicros[plan]
	if !ok || cap == 0 {
		return nil
	}
	if userID == nil {
		// Anon — relies on ratelimit.go check-count cap, not $ cap, since
		// we'd need IP-keyed aggregation here.
		return nil
	}
	var userSpend int
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_cost_micros), 0)
		   FROM checks
		  WHERE user_id = $1 AND created_at > NOW() - INTERVAL '24 hours'`,
		*userID,
	).Scan(&userSpend)
	if err != nil {
		return fmt.Errorf("budget: query user: %w", err)
	}
	if userSpend >= cap {
		return ErrUserOverBudget
	}
	return nil
}

// MicrosToUSD formats microcents as "$0.123" for log/error messages.
func MicrosToUSD(m int) string {
	return fmt.Sprintf("$%.3f", float64(m)/1_000_000)
}

func envDollarsToMicros(key string, defDollars float64) int {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return int(f * 1_000_000)
		}
	}
	return int(defDollars * 1_000_000)
}
