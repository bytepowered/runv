package runv

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	_diObjects    = make(map[string]reflect.Value, 4)
	_diPrototypes = make(map[string]func() reflect.Value, 4)
)

func diRegisterObject(obj interface{}) {
	objtype := fmt.Sprintf("%T", obj)
	fmt.Printf("[di]register object, type: %s\n", objtype)
	_diObjects[objtype] = reflect.ValueOf(obj)
}

func diRegisterProvider(providerFunc interface{}) {
	pftype := reflect.TypeOf(providerFunc)
	if pftype.Kind() != reflect.Func {
		panic(fmt.Sprintf("[di]register object provider func must be <function>, was: %T", providerFunc))
	}
	if pftype.NumOut() != 1 {
		panic(fmt.Sprintf("invalid return values of provider func, num: %d", pftype.NumOut()))
	}
	pfvalue := reflect.ValueOf(providerFunc)
	_diPrototypes[pftype.Out(0).String()] = func() reflect.Value {
		return pfvalue.Call(nil)[0]
	}
}

func diInjectDepens(obj interface{}) {
	intype := reflect.TypeOf(obj)
	invalue := reflect.ValueOf(obj)
	// 通过Setter函数注入
	for i := 0; i < intype.NumMethod(); i++ {
		typm := intype.Method(i)
		// SetCompA(CompA)这样的函数
		if typm.Type.NumOut() == 0 && !strings.HasPrefix(typm.Name, "Set") || typm.Type.NumIn() != 2 {
			continue
		}
		marg := typm.Type.In(1)
		if v, ok := _diload(marg.String()); ok {
			invalue.Method(i).Call(v)
		}
	}
	// TODO 通过结构体字段注入
}

func _diload(typekey string) ([]reflect.Value, bool) {
	// objects
	if ref, ok := _diObjects[typekey]; ok {
		return []reflect.Value{ref}, true
	}
	// by loader
	if provider, ok := _diPrototypes[typekey]; ok {
		ref := provider()
		diInjectDepens(ref.Interface())
		return []reflect.Value{ref}, true
	}
	return nil, false
}
