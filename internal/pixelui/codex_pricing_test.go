package pixelui

import "testing"

func TestCodexModelPriceFor(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    codexModelPrice
		exact   bool
	}{
		{name: "exact match", modelID: "gpt-5.1-codex", want: codexPricingTable["gpt-5.1-codex"], exact: true},
		{name: "case and whitespace insensitive", modelID: "  GPT-5.1-Codex  ", want: codexPricingTable["gpt-5.1-codex"], exact: true},
		{name: "dated suffix matches longest prefix", modelID: "gpt-5.1-codex-2026-01-15", want: codexPricingTable["gpt-5.1-codex"], exact: true},
		{name: "mini suffix does not match its non-mini prefix", modelID: "gpt-5.1-codex-mini-preview", want: codexPricingTable["gpt-5.1-codex-mini"], exact: true},
		{name: "unrecognized model falls back", modelID: "some-future-model", want: codexFallbackPrice, exact: false},
		{name: "empty model falls back", modelID: "", want: codexFallbackPrice, exact: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, exact := codexModelPriceFor(tt.modelID)
			if got != tt.want {
				t.Errorf("codexModelPriceFor(%q) price = %+v, want %+v", tt.modelID, got, tt.want)
			}
			if exact != tt.exact {
				t.Errorf("codexModelPriceFor(%q) exact = %v, want %v", tt.modelID, exact, tt.exact)
			}
		})
	}
}

func TestCodexEstimatedCostUSD(t *testing.T) {
	cost, exact := codexEstimatedCostUSD("gpt-5.1-codex", 250_000, 40_000)
	if !exact {
		t.Fatal("expected exact match for known model")
	}
	want := float64(250_000)/1_000_000*codexPricingTable["gpt-5.1-codex"].InputPerMillion +
		float64(40_000)/1_000_000*codexPricingTable["gpt-5.1-codex"].OutputPerMillion
	if cost != want {
		t.Errorf("cost = %v, want %v", cost, want)
	}

	if cost, exact := codexEstimatedCostUSD("", 0, 0); cost != 0 || exact {
		t.Errorf("empty model/zero tokens = (%v, %v), want (0, false)", cost, exact)
	}
}
