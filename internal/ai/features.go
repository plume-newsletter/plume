package ai

import (
	"context"
	"encoding/json"
	"strings"
)

// Insight is one dashboard insight card. The fields map directly to the
// dashboard's render: an optional bold lead-in (Strong), the body (Rest), and
// an optional muted suggestion (Muted).
type Insight struct {
	Strong string `json:"strong,omitempty"`
	Rest   string `json:"rest"`
	Muted  string `json:"muted,omitempty"`
}

// RuleCondition mirrors segment.Condition as plain JSON. The ai package stays
// decoupled from segment; the handler validates the output against the real
// compiler before trusting it.
type RuleCondition struct {
	Type  string `json:"type"`
	Op    string `json:"op"`
	Days  int    `json:"days,omitempty"`
	Field string `json:"field,omitempty"`
	Value string `json:"value,omitempty"`
}

// SegmentRules is the parsed "Ask AI" result for the segment builder.
type SegmentRules struct {
	Match      string          `json:"match"`
	Conditions []RuleCondition `json:"conditions"`
}

const insightsSystemPrompt = "You are a marketing analyst for the Plume email app. " +
	"You are given a JSON snapshot of a workspace's email analytics. " +
	"Return EXACTLY 3 short, specific, actionable insights grounded in the numbers. " +
	`Respond with ONLY a JSON array of 3 objects, each {"strong": string, "rest": string, "muted": string}. ` +
	"`strong` is a bold lead-in fragment (a metric or time, e.g. \"Tuesday 9 AM\" or \"312 subscribers\"). " +
	"`rest` continues the sentence. `muted` is a brief suggested next action. " +
	"No markdown, no code fences, no preamble — only the JSON array."

// Insights turns an analytics snapshot (JSON) into 3 dashboard insight cards.
func (s *Service) Insights(ctx context.Context, cfg Config, analyticsJSON string) ([]Insight, error) {
	if strings.TrimSpace(analyticsJSON) == "" {
		return nil, ErrEmpty
	}
	out, err := s.m.complete(ctx, cfg.APIKey, modelOr(cfg.Model), insightsSystemPrompt, "Analytics snapshot:\n\n"+analyticsJSON)
	if err != nil {
		return nil, err
	}
	var insights []Insight
	if err := json.Unmarshal([]byte(extractJSON(out)), &insights); err != nil {
		return nil, ErrRefused
	}
	if len(insights) == 0 {
		return nil, ErrRefused
	}
	if len(insights) > 3 {
		insights = insights[:3]
	}
	return insights, nil
}

const segmentRulesSystemPrompt = "You convert a plain-English audience description into Plume segment rules. " +
	`Respond with ONLY a JSON object {"match": "all"|"any", "conditions": [ ... ]}, no markdown or prose. ` +
	"Each condition is one of:\n" +
	`- {"type":"opened"|"clicked", "op":"in_last"|"ever"|"never", "days": N}  (days required only for in_last)` + "\n" +
	`- {"type":"status", "op":"is"|"is_not", "value":"active"|"pending"|"unsubscribed"}` + "\n" +
	`- {"type":"field", "op":"equals"|"not_equals"|"contains", "field":"<one of the available fields>", "value":"<text>"}` + "\n" +
	"Use ONLY the operators and values listed. For `field` conditions, use ONLY field names from the provided list; " +
	"if no field fits, omit that condition. Prefer the fewest conditions that capture the request."

// SegmentRules converts a natural-language audience description into segment
// conditions. availableFields are the workspace's custom-field names the model
// may reference. The returned rules are NOT yet validated — the caller must run
// them through the segment compiler.
func (s *Service) SegmentRules(ctx context.Context, cfg Config, prompt string, availableFields []string) (SegmentRules, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return SegmentRules{}, ErrEmpty
	}
	if len(prompt) > maxInputChars {
		return SegmentRules{}, ErrTooLong
	}
	fields := "(none)"
	if len(availableFields) > 0 {
		fields = strings.Join(availableFields, ", ")
	}
	user := "Available custom fields: " + fields + "\n\nDescription: " + prompt
	out, err := s.m.complete(ctx, cfg.APIKey, modelOr(cfg.Model), segmentRulesSystemPrompt, user)
	if err != nil {
		return SegmentRules{}, err
	}
	var rules SegmentRules
	if err := json.Unmarshal([]byte(extractJSON(out)), &rules); err != nil {
		return SegmentRules{}, ErrRefused
	}
	if rules.Match != "all" && rules.Match != "any" {
		rules.Match = "all"
	}
	if len(rules.Conditions) == 0 {
		return SegmentRules{}, ErrRefused
	}
	return rules, nil
}

func modelOr(m string) string {
	if m == "" {
		return DefaultModel
	}
	return m
}

// extractJSON pulls the JSON payload out of a model reply, tolerating ```json
// code fences or stray prose around it by slicing to the outermost braces.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	// Slice from the first array/object opener to its matching last closer.
	start := strings.IndexAny(s, "[{")
	if start < 0 {
		return s
	}
	open := s[start]
	close := byte('}')
	if open == '[' {
		close = ']'
	}
	end := strings.LastIndexByte(s, close)
	if end < start {
		return s
	}
	return s[start : end+1]
}
