package runv

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	_diTypedInstances = make(map[reflect.Type]interface{}, 4)
	_diTypedProviders = make(map[reflect.Type]func() interface{}, 4)
)

func diRegisterInstance(inst interface{}) {
	typ := reflect.TypeOf(inst)
	fmt.Printf("[di]register instance, type: %s\n", typ.String())
	_diTypedInstances[typ] = inst
}

func diRegisterProvider(provider interface{}) {
	typ := reflect.TypeOf(provider)
	if typ.Kind() != reflect.Func {
		panic(fmt.Sprintf("[di]register object provider func must be <function>, was: %T", provider))
	}
	if typ.NumOut() != 1 {
		panic(fmt.Sprintf("invalid return values of provider func, num: %d", typ.NumOut()))
	}
	val := reflect.ValueOf(provider)
	_diTypedProviders[typ] = func() interface{} {
		return val.Call(nil)[0].Interface()
	}
}

func diInjectDepens(hostobj interface{}) {
	meta := reflect.TypeOf(hostobj)
	invoker := reflect.ValueOf(hostobj)
	// 通过Setter函数注入
	for i := 0; i < meta.NumMethod(); i++ {
		mthtype := meta.Method(i)
		// SetCompA(CompA), InjectComB(CompB)这样的函数
		if mthtype.Type.NumOut() != 0 || mthtype.Type.NumIn() != 2 {
			continue
		}
		if !strings.HasPrefix(mthtype.Name, "Set") && !strings.HasPrefix(mthtype.Name, "Inject") {
			continue
		}
		argtype := mthtype.Type.In(1)
		switch argtype.Kind() {
		case reflect.Ptr:
			if obj, ok := diLoadInstanceByType(argtype); ok {
				invoker.Method(i).Call([]reflect.Value{reflect.ValueOf(obj)})
			}

		//case reflect.Slice:
		//	if argtype.Elem().Kind() == reflect.Interface {
		//		eletype := argtype.Elem()
		//		objs, ok := diLoadInstanceByInterface(eletype)
		//		if !ok {
		//			continue
		//		}
		//		fmt.Printf("Objes Type: %T, %s\n", objs, objs)
		//		args := reflect.MakeSlice(reflect.SliceOf(eletype), 0, len(objs))
		//		for _, obj := range objs {
		//			fmt.Printf("Obj: %T, %s\n", obj, obj)
		//			//nv := reflect.New(eletype)
		//			//nv.Set(reflect.ValueOf(obj))
		//			//args = reflect.AppendSlice(args, nv)
		//		}
		//		invoker.Method(i).Call([]reflect.Value{args})
		//	}
		}
	}
	// TODO 通过结构体字段注入
}

func diLoadInstanceByType(typ reflect.Type) (interface{}, bool) {
	// instances
	if ref, ok := _diTypedInstances[typ]; ok {
		return ref, true
	}
	// by provider
	if provider, ok := _diTypedProviders[typ]; ok {
		obj := provider()
		diInjectDepens(obj)
		return obj, true
	}
	return nil, false
}

func diLoadInstanceByInterface(iface reflect.Type) (out []interface{}, ok bool) {
	// instances
	for typ, inst := range _diTypedInstances {
		if typ.Implements(iface) {
			out = append(out, inst)
		}
	}
	// providers
	for typ, provider := range _diTypedProviders {
		if typ.Implements(iface) {
			obj := provider()
			diInjectDepens(obj)
			out = append(out, obj)
		}
	}
	return out, true
}
