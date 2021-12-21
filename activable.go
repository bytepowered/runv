package runv

import (
	"os"
)

type ActiveExprs map[string]interface{}

func (ex ActiveExprs) IsActive() bool {
	// { DEPLOY_ENV = ["DEV", "UAT", "PROD"] }
	// { DEPLOY_ENV = ["DEV", "UAT", "PROD"] }
	for key, expr := range ex {
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
			if !ex.matches(env, expr.([]string)) {
				return false
			}
		}
	}
	return true
}

func (ex ActiveExprs) matches(in string, exprs []string) bool {
	for _, v := range exprs {
		if in == v {
			return true
		}
	}
	return false
}
