package runv

import (
	"context"
	"time"
)

var _ Context = new(VarContext)

type VarContext struct {
	ctx  context.Context
	vars map[interface{}]interface{}
}

func NewVarContext(ctx context.Context, vars map[interface{}]interface{}) *VarContext {
	if vars == nil {
		vars = make(map[interface{}]interface{}, 0)
	}
	return &VarContext{ctx: ctx, vars: vars}
}

func NewVarContextWith(ctx context.Context, vars map[interface{}]interface{}) *VarContext {
	return NewVarContext(ctx, vars)
}

func NewVarContext0(ctx context.Context) *VarContext {
	return NewVarContext(ctx, nil)
}

func (s *VarContext) Deadline() (deadline time.Time, ok bool) {
	return s.ctx.Deadline()
}

func (s *VarContext) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *VarContext) Err() error {
	return s.ctx.Err()
}

func (s *VarContext) Value(key interface{}) interface{} {
	v, _ := s.ValueE(key)
	return v
}

func (s *VarContext) ValueE(key interface{}) (value interface{}, ok bool) {
	value, ok = s.vars[key]
	if ok {
		return value, true
	}
	value = s.ctx.Value(key)
	return value, nil == value
}
