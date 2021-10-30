package runv

import (
	"reflect"
	"strings"
)

func inject(in interface{}, vloader func(typ string) ([]reflect.Value, bool)) {
	intype := reflect.TypeOf(in)
	invalue := reflect.ValueOf(in)
	// 通过Setter函数注入
	for i := 0; i < intype.NumMethod(); i++ {
		typm := intype.Method(i)
		// SetCompA(CompA)这样的函数
		if !strings.HasPrefix(typm.Name, "Set") || typm.Type.NumIn() != 2 {
			continue
		}
		marg := typm.Type.In(1)
		if v, ok := vloader(marg.String()); ok {
			invalue.Method(i).Call(v)
		}
	}
	// TODO 通过结构体字段注入
}
