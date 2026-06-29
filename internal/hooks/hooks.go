// Package hooks is Plume's extensibility kernel: WordPress-style named Actions
// (fire-and-react) and Filters (value transforms), each run in priority order.
package hooks

import (
	"context"
	"fmt"
	"sort"
)

type ActionFunc func(ctx context.Context, payload any) error
type FilterFunc func(ctx context.Context, value any) (any, error)

type actionReg struct {
	priority int
	fn       ActionFunc
}
type filterReg struct {
	priority int
	fn       FilterFunc
}

// Hooks holds registered actions and filters keyed by name. Register at startup;
// it is not safe for concurrent registration after the server is serving.
type Hooks struct {
	actions map[string][]actionReg
	filters map[string][]filterReg
}

func New() *Hooks {
	return &Hooks{
		actions: map[string][]actionReg{},
		filters: map[string][]filterReg{},
	}
}

func (h *Hooks) AddAction(name string, priority int, fn ActionFunc) {
	regs := append(h.actions[name], actionReg{priority, fn})
	sort.SliceStable(regs, func(i, j int) bool { return regs[i].priority < regs[j].priority })
	h.actions[name] = regs
}

// DoAction runs every handler for name in ascending priority order, stopping at
// the first error.
func (h *Hooks) DoAction(ctx context.Context, name string, payload any) error {
	for _, r := range h.actions[name] {
		if err := r.fn(ctx, payload); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hooks) AddFilter(name string, priority int, fn FilterFunc) {
	regs := append(h.filters[name], filterReg{priority, fn})
	sort.SliceStable(regs, func(i, j int) bool { return regs[i].priority < regs[j].priority })
	h.filters[name] = regs
}

// ApplyFilter threads value through every handler for name in ascending priority
// order, returning the transformed value.
func (h *Hooks) ApplyFilter(ctx context.Context, name string, value any) (any, error) {
	for _, r := range h.filters[name] {
		out, err := r.fn(ctx, value)
		if err != nil {
			return nil, err
		}
		value = out
	}
	return value, nil
}

// Filter is a type-safe wrapper over ApplyFilter for core call sites.
func Filter[T any](ctx context.Context, h *Hooks, name string, value T) (T, error) {
	out, err := h.ApplyFilter(ctx, name, value)
	if err != nil {
		var zero T
		return zero, err
	}
	typed, ok := out.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("hooks: filter %q returned %T, want %T", name, out, zero)
	}
	return typed, nil
}
