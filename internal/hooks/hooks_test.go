package hooks

import (
	"context"
	"testing"
)

func TestDoActionRunsAllHandlersInPriorityOrder(t *testing.T) {
	h := New()
	var order []string
	h.AddAction("thing.happened", 20, func(_ context.Context, _ any) error {
		order = append(order, "b")
		return nil
	})
	h.AddAction("thing.happened", 10, func(_ context.Context, _ any) error {
		order = append(order, "a")
		return nil
	})

	if err := h.DoAction(context.Background(), "thing.happened", nil); err != nil {
		t.Fatalf("DoAction: %v", err)
	}
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("order = %v, want [a b]", order)
	}
}

func TestApplyFilterChainsInPriorityOrder(t *testing.T) {
	h := New()
	// Registered B(20) before A(10); priority must force A before B.
	h.AddFilter("render", 20, func(_ context.Context, v any) (any, error) {
		return v.(string) + "B", nil
	})
	h.AddFilter("render", 10, func(_ context.Context, v any) (any, error) {
		return v.(string) + "A", nil
	})

	out, err := Filter(context.Background(), h, "render", "start:")
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}
	if out != "start:AB" {
		t.Fatalf("out = %q, want %q", out, "start:AB")
	}
}

func TestApplyFilterReturnsInputWhenNoFilters(t *testing.T) {
	h := New()
	out, err := Filter(context.Background(), h, "nope", "untouched")
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}
	if out != "untouched" {
		t.Fatalf("out = %q, want %q", out, "untouched")
	}
}

func TestFilterReturnsErrorOnTypeMismatch(t *testing.T) {
	h := New()
	// A filter registered under "render" that returns an int, while the caller asks for string.
	h.AddFilter("render", 10, func(_ context.Context, _ any) (any, error) {
		return 42, nil
	})

	_, err := Filter[string](context.Background(), h, "render", "start:")
	if err == nil {
		t.Fatal("expected a type-mismatch error, got nil")
	}
}
