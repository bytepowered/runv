package runv

import (
	"context"
	"time"
)

var _ Context = new(ContextV)

type ContextV struct {
	ctx  context.Context
	vars map[interface{}]interface{}
}

func NewContextV(ctx context.Context, vars map[interface{}]interface{}) *ContextV {
	if vars == nil {
		vars = make(map[interface{}]interface{}, 0)
	}
	return &ContextV{ctx: ctx, vars: vars}
}

func NewContextVX(ctx context.Context, vars map[interface{}]interface{}) *ContextV {
	return NewContextV(ctx, vars)
}

func NewContextV0(ctx context.Context) *ContextV {
	return NewContextV(ctx, nil)
}

func (s *ContextV) Deadline() (deadline time.Time, ok bool) {
	return s.ctx.Deadline()
}

func (s *ContextV) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *ContextV) Err() error {
	return s.ctx.Err()
}

func (s *ContextV) Value(key interface{}) interface{} {
	v, _ := s.ValueE(key)
	return v
}

func (s *ContextV) ValueE(key interface{}) (value interface{}, ok bool) {
	value, ok = s.vars[key]
	if ok {
		return value, true
	}
	value = s.ctx.Value(key)
	return value, nil == value
}
