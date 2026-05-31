package budget_test

import (
	"testing"

	"github.com/zeroroot-ai/gibson-redteam-example/llm-redteam/internal/budget"
)

func TestLimitsExceeded(t *testing.T) {
	tests := []struct {
		name          string
		limits        budget.Limits
		turns, tokens int
		want          bool
	}{
		{"unlimited", budget.Limits{}, 1000, 1_000_000, false},
		{"under turns", budget.Limits{MaxTurns: 10}, 9, 0, false},
		{"at turns", budget.Limits{MaxTurns: 10}, 10, 0, true},
		{"over tokens", budget.Limits{MaxTokens: 100}, 0, 150, true},
		{"only token limit active", budget.Limits{MaxTokens: 100}, 9999, 50, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.limits.Exceeded(tt.turns, tt.tokens); got != tt.want {
				t.Errorf("Exceeded(%d,%d) = %v, want %v", tt.turns, tt.tokens, got, tt.want)
			}
		})
	}
}

func TestOutcome(t *testing.T) {
	tests := []struct {
		name      string
		exceeded  bool
		completed bool
		probesRun int
		want      budget.Status
	}{
		{"completed wins", false, true, 5, budget.StatusSuccess},
		{"completed even if exceeded", true, true, 5, budget.StatusSuccess},
		{"exceeded with work is partial", true, false, 3, budget.StatusPartial},
		{"exceeded with no work is budget_exceeded", true, false, 0, budget.StatusBudgetExceeded},
		{"early stop is partial", false, false, 2, budget.StatusPartial},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := budget.Outcome(tt.exceeded, tt.completed, tt.probesRun); got != tt.want {
				t.Errorf("Outcome(%v,%v,%d) = %q, want %q", tt.exceeded, tt.completed, tt.probesRun, got, tt.want)
			}
		})
	}
}
