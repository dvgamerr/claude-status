package pixelui

import "strings"

// codexModelPrice is USD per 1,000,000 tokens. Providers charge output
// tokens at a different (usually higher) rate than input tokens, so the
// two are tracked separately rather than a single blended rate.
type codexModelPrice struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// codexPricingTable holds approximate published per-model rates for
// OpenAI/Codex models, in USD per 1M tokens. Codex reports its own model
// string verbatim (see internal/codex), so entries are matched by exact
// string first, then by longest known prefix (Codex often appends a dated
// or "-mini"/"-codex" suffix to a base family name). These rates will drift
// from OpenAI's current published pricing over time and should be updated
// when they do — this is an estimate, not a billed amount, matching the
// same disclaimer already given for Claude's own reported cost.
var codexPricingTable = map[string]codexModelPrice{
	"gpt-5.1-codex-mini": {InputPerMillion: 0.25, OutputPerMillion: 2.00},
	"gpt-5.1-codex":      {InputPerMillion: 1.25, OutputPerMillion: 10.00},
	"gpt-5-codex":        {InputPerMillion: 1.25, OutputPerMillion: 10.00},
	"gpt-5.1":            {InputPerMillion: 1.25, OutputPerMillion: 10.00},
	"gpt-5-mini":         {InputPerMillion: 0.25, OutputPerMillion: 2.00},
	"gpt-5-nano":         {InputPerMillion: 0.05, OutputPerMillion: 0.40},
	"gpt-5":              {InputPerMillion: 1.25, OutputPerMillion: 10.00},
	"o4-mini":            {InputPerMillion: 1.10, OutputPerMillion: 4.40},
	"o3":                 {InputPerMillion: 2.00, OutputPerMillion: 8.00},
	"gpt-4.1-mini":       {InputPerMillion: 0.40, OutputPerMillion: 1.60},
	"gpt-4.1":            {InputPerMillion: 2.00, OutputPerMillion: 8.00},
	"gpt-4o-mini":        {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"gpt-4o":             {InputPerMillion: 2.50, OutputPerMillion: 10.00},
}

// codexFallbackPrice estimates spend for a model ID that matches nothing in
// codexPricingTable (an unrecognized or future model), using the current
// flagship-tier rate so an unknown model still shows a rough number instead
// of nothing.
var codexFallbackPrice = codexModelPrice{InputPerMillion: 1.25, OutputPerMillion: 10.00}

// codexModelPriceFor resolves pricing for a Codex-reported model ID. The
// bool reports whether the ID matched a known table entry (exact or
// prefix) rather than the generic fallback rate.
func codexModelPriceFor(modelID string) (codexModelPrice, bool) {
	id := strings.ToLower(strings.TrimSpace(modelID))
	if id == "" {
		return codexFallbackPrice, false
	}
	if price, ok := codexPricingTable[id]; ok {
		return price, true
	}
	var bestKey string
	for key := range codexPricingTable {
		if strings.HasPrefix(id, key) && len(key) > len(bestKey) {
			bestKey = key
		}
	}
	if bestKey != "" {
		return codexPricingTable[bestKey], true
	}
	return codexFallbackPrice, false
}

// codexEstimatedCostUSD estimates spend from token counts and the model's
// per-million rates. exact reports whether modelID matched a known pricing
// entry rather than the generic fallback rate.
func codexEstimatedCostUSD(modelID string, inputTokens, outputTokens int64) (cost float64, exact bool) {
	price, exact := codexModelPriceFor(modelID)
	cost = float64(inputTokens)/1_000_000*price.InputPerMillion + float64(outputTokens)/1_000_000*price.OutputPerMillion
	return cost, exact
}
