package inject

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	typedObjects   = make(map[key]interface{}, 4)
	typedProviders = make(map[key]func() interface{}, 4)
)

type key struct {
	t    reflect.Type
	name string
}

func (k key) String() string {
	return fmt.Sprintf("type: %s, name: %s", k.t, k.name)
}

func RegisterObject(inst interface{}) {
	typ := reflect.TypeOf(inst)
	fmt.Printf("[DI]register object, type: %s\n", typ.String())
	typedObjects[key{t: typ, name: typ.Name()}] = inst
}

func RegisterProvider(provider interface{}) {
	protyp := reflect.TypeOf(provider)
	if protyp.Kind() != reflect.Func {
		panic(fmt.Sprintf("[DI]register object-provider func must be <function>, was: %T", provider))
	}
	if protyp.NumOut() != 1 {
		panic(fmt.Sprintf("invalid return values of provider func, num: %d", protyp.NumOut()))
	}
	proval := reflect.ValueOf(provider)
	elem := protyp.Out(0)
	typedProviders[key{t: elem, name: elem.Name()}] = func() interface{} {
		return proval.Call(nil)[0].Interface()
	}
}

func ResolveDeps(hostobj interface{}) {
	meta := reflect.TypeOf(hostobj)
	invoker := reflect.ValueOf(hostobj)
	doMethodInject(meta, invoker)
	// TODO 通过结构体字段注入
}

func doMethodInject(meta reflect.Type, invoker reflect.Value) {
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
			if obj, ok := LoadObjectByType(argtype); ok {
				invoker.Method(i).Call([]reflect.Value{reflect.ValueOf(obj)})
			}

		case reflect.Slice:
			if argtype.Elem().Kind() != reflect.Interface {
				continue
			}
			eletype := argtype.Elem()
			objs, ok := LoadObjectsByIface(eletype)
			if !ok {
				continue
			}
			args := reflect.MakeSlice(reflect.SliceOf(eletype), len(objs), len(objs))
			for at, obj := range objs {
				args.Index(at).Set(reflect.ValueOf(obj))
			}
			invoker.Method(i).Call([]reflect.Value{args})
		}
	}
}

func LoadObjectByType(typ reflect.Type) (interface{}, bool) {
	// instances
	k := key{t: typ, name: typ.Name()}
	if ref, ok := typedObjects[k]; ok {
		return ref, true
	}
	// by provider
	if provider, ok := typedProviders[k]; ok {
		obj := provider()
		ResolveDeps(obj)
		return obj, true
	}
	return nil, false
}

func LoadObjectsByIface(iface reflect.Type) (out []interface{}, ok bool) {
	// instances
	for k, inst := range typedObjects {
		if k.t.Implements(iface) {
			out = append(out, inst)
		}
	}
	// providers
	for k, provider := range typedProviders {
		if k.t.Implements(iface) {
			obj := provider()
			ResolveDeps(obj)
			out = append(out, obj)
		}
	}
	return out, true
}
