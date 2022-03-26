package runv

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestActiveExprsMulti(t *testing.T) {
	exprs := EnvActiveExprs{
		"DEPLOY": []string{"UAT", "PROD"},
	}
	_ = os.Setenv("DEPLOY", "DEV")
	assert.False(t, exprs.IsActive())

	_ = os.Setenv("DEPLOY", "UAT")
	assert.True(t, exprs.IsActive())

	_ = os.Setenv("DEPLOY", "PROD")
	assert.True(t, exprs.IsActive())
}

func TestActiveExprsSingle(t *testing.T) {
	exprs := EnvActiveExprs{
		"DEPLOY": "PROD",
		"APP":    "RUNV",
	}
	_ = os.Setenv("DEPLOY", "DEV")
	assert.False(t, exprs.IsActive())

	_ = os.Setenv("DEPLOY", "UAT")
	assert.False(t, exprs.IsActive())

	_ = os.Setenv("DEPLOY", "PROD")
	assert.False(t, exprs.IsActive())

	_ = os.Setenv("APP", "RUNV")
	assert.True(t, exprs.IsActive())
}
