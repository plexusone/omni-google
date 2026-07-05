package gemini

import (
	"testing"

	"github.com/plexusone/omnillm-core/provider"
)

func TestConvertThinkingConfig_ReasoningEffort(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name            string
		reasoningEffort string
		wantLevel       string
	}{
		{"none maps to MINIMAL", provider.ReasoningEffortNone, ThinkingLevelMinimal},
		{"low maps to LOW", provider.ReasoningEffortLow, ThinkingLevelLow},
		{"medium maps to MEDIUM", provider.ReasoningEffortMedium, ThinkingLevelMedium},
		{"high maps to HIGH", provider.ReasoningEffortHigh, ThinkingLevelHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effort := tt.reasoningEffort
			req := &provider.ChatCompletionRequest{
				ReasoningEffort: &effort,
			}

			got := p.convertThinkingConfig(req)

			if got == nil {
				t.Fatal("convertThinkingConfig() returned nil")
			}
			if got.ThinkingLevel == nil {
				t.Fatal("ThinkingLevel is nil")
			}
			if *got.ThinkingLevel != tt.wantLevel {
				t.Errorf("ThinkingLevel = %q, want %q", *got.ThinkingLevel, tt.wantLevel)
			}
		})
	}
}

func TestConvertThinkingConfig_ThinkingType(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name      string
		thinkType string
		wantLevel string
	}{
		{"enabled maps to HIGH", provider.ThinkingTypeEnabled, ThinkingLevelHigh},
		{"disabled maps to MINIMAL", provider.ThinkingTypeDisabled, ThinkingLevelMinimal},
		{"adaptive maps to MEDIUM", provider.ThinkingTypeAdaptive, ThinkingLevelMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &provider.ChatCompletionRequest{
				Thinking: &provider.ThinkingConfig{
					Type: tt.thinkType,
				},
			}

			got := p.convertThinkingConfig(req)

			if got == nil {
				t.Fatal("convertThinkingConfig() returned nil")
			}
			if got.ThinkingLevel == nil {
				t.Fatal("ThinkingLevel is nil")
			}
			if *got.ThinkingLevel != tt.wantLevel {
				t.Errorf("ThinkingLevel = %q, want %q", *got.ThinkingLevel, tt.wantLevel)
			}
		})
	}
}

func TestConvertThinkingConfig_BudgetTokens(t *testing.T) {
	p := &Provider{}

	budget := int64(8192)
	req := &provider.ChatCompletionRequest{
		Thinking: &provider.ThinkingConfig{
			Type:         provider.ThinkingTypeEnabled,
			BudgetTokens: &budget,
		},
	}

	got := p.convertThinkingConfig(req)

	if got == nil {
		t.Fatal("convertThinkingConfig() returned nil")
	}
	if got.ThinkingBudget == nil {
		t.Fatal("ThinkingBudget is nil")
	}
	if *got.ThinkingBudget != int32(budget) {
		t.Errorf("ThinkingBudget = %d, want %d", *got.ThinkingBudget, budget)
	}
}

func TestConvertThinkingConfig_ThinkingPriority(t *testing.T) {
	p := &Provider{}

	// When both Thinking and ReasoningEffort are set, Thinking takes priority
	effort := provider.ReasoningEffortLow
	req := &provider.ChatCompletionRequest{
		ReasoningEffort: &effort,
		Thinking: &provider.ThinkingConfig{
			Type: provider.ThinkingTypeEnabled,
		},
	}

	got := p.convertThinkingConfig(req)

	if got == nil {
		t.Fatal("convertThinkingConfig() returned nil")
	}
	if got.ThinkingLevel == nil {
		t.Fatal("ThinkingLevel is nil")
	}
	// Should use Thinking (HIGH) not ReasoningEffort (LOW)
	if *got.ThinkingLevel != ThinkingLevelHigh {
		t.Errorf("ThinkingLevel = %q, want %q (Thinking should take priority)", *got.ThinkingLevel, ThinkingLevelHigh)
	}
}

func TestConvertThinkingConfig_Nil(t *testing.T) {
	p := &Provider{}

	req := &provider.ChatCompletionRequest{
		// No reasoning config
	}

	got := p.convertThinkingConfig(req)

	if got != nil {
		t.Errorf("convertThinkingConfig() = %v, want nil when no reasoning config", got)
	}
}
