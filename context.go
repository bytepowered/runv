package runv

import (
	"context"
	"github.com/sirupsen/logrus"
)

var _ Context = new(StateContext)

type StateContext struct {
	ctx    context.Context
	vars   map[interface{}]interface{}
	logger *logrus.Logger
}

func newStateContext(ctx context.Context, logger *logrus.Logger, vars map[interface{}]interface{}) *StateContext {
	if vars == nil {
		vars = make(map[interface{}]interface{}, 8)
	}
	return &StateContext{ctx: ctx, logger: logger, vars: vars}
}

func (s *StateContext) Context() context.Context {
	return s.ctx
}

func (s *StateContext) GetVarE(key interface{}) (value interface{}, ok bool) {
	value, ok = s.vars[key]
	if ok {
		return value, true
	}
	value = s.ctx.Value(key)
	return value, nil == value
}

func (s *StateContext) GetVar(key interface{}) (value interface{}) {
	value, _ = s.GetVarE(key)
	return
}

func (s *StateContext) Log() *logrus.Logger {
	return s.logger
}
