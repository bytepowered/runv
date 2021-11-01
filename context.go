package runv

import (
	"context"
	"github.com/sirupsen/logrus"
	"time"
)

var _ Context = new(StateContext)

type StateContext struct {
	ctx    context.Context
	vars   map[interface{}]interface{}
	logger *logrus.Logger
}

func NewStateContext(ctx context.Context, logger *logrus.Logger, vars map[interface{}]interface{}) *StateContext {
	if vars == nil {
		vars = make(map[interface{}]interface{}, 0)
	}
	return &StateContext{ctx: ctx, logger: logger, vars: vars}
}

func NewStateContext1(ctx context.Context, vars map[interface{}]interface{}) *StateContext {
	return NewStateContext(ctx, nil, vars)
}

func NewStateContext2(ctx context.Context) *StateContext {
	return NewStateContext(ctx, nil, nil)
}

func (s *StateContext) Deadline() (deadline time.Time, ok bool) {
	return s.ctx.Deadline()
}

func (s *StateContext) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *StateContext) Err() error {
	return s.ctx.Err()
}

func (s *StateContext) Value(key interface{}) interface{} {
	v, _ := s.ValueE(key)
	return v
}

func (s *StateContext) ValueE(key interface{}) (value interface{}, ok bool) {
	value, ok = s.vars[key]
	if ok {
		return value, true
	}
	value = s.ctx.Value(key)
	return value, nil == value
}

func (s *StateContext) Log() *logrus.Logger {
	return s.logger
}
