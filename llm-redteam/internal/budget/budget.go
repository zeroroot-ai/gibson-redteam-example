// Package budget tracks a campaign's turn and token usage against its
// limits and decides the terminal result status. It is pure — no LLM, no
// harness — so the agent's loop can consult it deterministically.
package budget

// Status is the terminal outcome of a campaign loop. The values mirror
// the SDK's agent result statuses.
type Status string

const (
	StatusSuccess        Status = "success"
	StatusPartial        Status = "partial"
	StatusBudgetExceeded Status = "budget_exceeded"
)

// Limits caps a campaign. A zero (or negative) value means "unlimited"
// for that dimension.
type Limits struct {
	MaxTurns  int
	MaxTokens int
}

// Exceeded reports whether usage has reached or passed any active limit.
func (l Limits) Exceeded(turns, tokens int) bool {
	if l.MaxTurns > 0 && turns >= l.MaxTurns {
		return true
	}
	if l.MaxTokens > 0 && tokens >= l.MaxTokens {
		return true
	}
	return false
}

// Outcome maps a loop's end state to a terminal status:
//   - completed all planned work        → success
//   - stopped on budget, having run probes → partial
//   - stopped on budget before any probe → budget_exceeded
//   - stopped early for another reason   → partial
func Outcome(exceeded, completed bool, probesRun int) Status {
	switch {
	case completed:
		return StatusSuccess
	case exceeded && probesRun == 0:
		return StatusBudgetExceeded
	default:
		return StatusPartial
	}
}
