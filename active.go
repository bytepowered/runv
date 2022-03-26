package runv

import (
	"os"
)

type EnvActiveExprs map[string]interface{}

func (e EnvActiveExprs) IsActive() bool {
	// { DEPLOY_ENV = ["DEV", "UAT", "PROD"] }
	// { DEPLOY_ENV = ["DEV", "UAT", "PROD"] }
	for key, expr := range e {
		env, ok := os.LookupEnv(key)
		if !ok {
			return false
		}
		switch expr.(type) {
		case string:
			if env != expr.(string) {
				return false
			}
		case []string:
			if !e.matches(env, expr.([]string)) {
				return false
			}
		}
	}
	return true
}

func (e EnvActiveExprs) matches(in string, exprs []string) bool {
	for _, v := range exprs {
		if in == v {
			return true
		}
	}
	return false
}
