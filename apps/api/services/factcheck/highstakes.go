package factcheck

import (
	"regexp"
	"strings"
)

// highStakesPatterns are subject areas where an early-exit shortcut is
// never acceptable — the cost of a wrong "verified" verdict on these is
// disproportionately high. The screener may still produce a hint, but
// the full fanout-and-judge pipeline always runs.
var highStakesPatterns = []*regexp.Regexp{
	// Public health / medical
	regexp.MustCompile(`(?i)\b(vaccin|mRNA|covid|sars-?cov|fluoride|chemtrail|fluoride|autism|measles|chemotherapy|insulin)`),
	// Elections / political process
	regexp.MustCompile(`(?i)\b(electio|ballot|voter ?fraud|polling|recount|inauguration|impeach)`),
	// Active conflicts / casualty counts
	regexp.MustCompile(`(?i)\b(casualt|killed|wounded|airstrike|massacre|war crime|genocide)`),
	// Finance — moving claims about specific tickers
	regexp.MustCompile(`\$[A-Z]{1,5}\b`),
	regexp.MustCompile(`(?i)\b(market crash|recession|inflation|interest rate|federal reserve|sec investig)`),
	// Climate science
	regexp.MustCompile(`(?i)\b(climate|global warming|ipcc|sea level|carbon emission)`),
	// Names of public figures with frequent misinformation
	regexp.MustCompile(`(?i)\b(biden|trump|harris|musk|putin|netanyahu|zelensk)`),
}

// IsHighStakes returns true if the claim matches any high-stakes pattern.
// The screener will mark such claims with high_stakes=true so the
// orchestrator skips any early-exit code path.
func IsHighStakes(claim string) bool {
	s := strings.ToLower(claim)
	for _, re := range highStakesPatterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}
