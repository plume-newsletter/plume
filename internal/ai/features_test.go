package ai

import (
	"context"
	"testing"
)

func TestExtractJSON(t *testing.T) {
	cases := map[string]string{
		`{"a":1}`:                         `{"a":1}`,
		"```json\n{\"a\":1}\n```":         `{"a":1}`,
		"Here you go:\n```\n[1,2,3]\n```": `[1,2,3]`,
		"prose [1] more":                  `[1]`,
		"no json here":                    "no json here",
	}
	for in, want := range cases {
		if got := extractJSON(in); got != want {
			t.Errorf("extractJSON(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInsightsParsesArray(t *testing.T) {
	stub := &stubMessager{reply: "```json\n[{\"strong\":\"Tue 9 AM\",\"rest\":\" is your best slot.\",\"muted\":\" Schedule then.\"}]\n```"}
	svc := New(stub)
	out, err := svc.Insights(context.Background(), Config{APIKey: "k"}, `{"subscribers":10}`)
	if err != nil {
		t.Fatalf("Insights: %v", err)
	}
	if len(out) != 1 || out[0].Strong != "Tue 9 AM" {
		t.Fatalf("out = %+v", out)
	}
}

func TestInsightsRefusesGarbage(t *testing.T) {
	svc := New(&stubMessager{reply: "I can't do that"})
	if _, err := svc.Insights(context.Background(), Config{APIKey: "k"}, `{}`); err != ErrRefused {
		t.Fatalf("err = %v, want ErrRefused", err)
	}
}

func TestSegmentRulesParsesAndDefaultsMatch(t *testing.T) {
	stub := &stubMessager{reply: `{"match":"bogus","conditions":[{"type":"status","op":"is","value":"active"}]}`}
	svc := New(stub)
	out, err := svc.SegmentRules(context.Background(), Config{APIKey: "k"}, "active people", nil)
	if err != nil {
		t.Fatalf("SegmentRules: %v", err)
	}
	if out.Match != "all" { // invalid match coerced to "all"
		t.Fatalf("match = %q", out.Match)
	}
	if len(out.Conditions) != 1 || out.Conditions[0].Type != "status" {
		t.Fatalf("conditions = %+v", out.Conditions)
	}
}
