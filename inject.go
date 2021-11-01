package inject

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	typedObjects   = make(map[objectkey]interface{}, 4)
	typedProviders = make(map[objectkey]func() interface{}, 4)
)

type objectkey struct {
	typ  reflect.Type
	name string
}

func (k objectkey) String() string {
	return fmt.Sprintf("type: %s, name: %s", k.typ, k.name)
}

func RegisterObject(obj interface{}) {
	typ := reflect.TypeOf(obj)
	typedObjects[objectkey{typ: typ, name: typ.Name()}] = obj
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
	typedProviders[objectkey{typ: elem, name: elem.Name()}] = func() interface{} {
		return proval.Call(nil)[0].Interface()
	}
}

func ResolveDeps(host interface{}) {
	meta := reflect.TypeOf(host)
	invoker := reflect.ValueOf(host)
	switch meta.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Struct:
		injectSetter(meta, invoker)
	}
	// TODO 通过结构体字段注入
}

func injectSetter(meta reflect.Type, invoker reflect.Value) {
	// 通过Setter函数注入
	for i := 0; i < meta.NumMethod(); i++ {
		mType := meta.Method(i)
		// SetCompA(CompA), InjectComB(CompB)这样的函数
		if mType.Type.NumOut() != 0 || mType.Type.NumIn() != 2 {
			continue
		}
		if !strings.HasPrefix(mType.Name, "Set") && !strings.HasPrefix(mType.Name, "Inject") {
			continue
		}
		aType := mType.Type.In(1)
		switch aType.Kind() {
		case reflect.Ptr:
			if obj, ok := LoadObjectByType(aType); ok {
				invoker.Method(i).Call([]reflect.Value{reflect.ValueOf(obj)})
			}

		case reflect.Slice:
			if aType.Elem().Kind() != reflect.Interface {
				continue
			}
			eType := aType.Elem()
			objs, ok := LoadObjectsByIface(eType)
			if !ok {
				continue
			}
			args := reflect.MakeSlice(reflect.SliceOf(eType), len(objs), len(objs))
			for at, obj := range objs {
				args.Index(at).Set(reflect.ValueOf(obj))
			}
			invoker.Method(i).Call([]reflect.Value{args})
		}
	}
}

func LoadObjectByType(typ reflect.Type) (interface{}, bool) {
	// instances
	key := objectkey{typ: typ, name: typ.Name()}
	if ref, ok := typedObjects[key]; ok {
		return ref, true
	}
	// by provider
	if provider, ok := typedProviders[key]; ok {
		obj := provider()
		ResolveDeps(obj)
		return obj, true
	}
	return nil, false
}

func LoadObjectsByIface(iface reflect.Type) (out []interface{}, ok bool) {
	// instances
	for k, inst := range typedObjects {
		if k.typ.Implements(iface) {
			out = append(out, inst)
		}
	}
	// providers
	for k, provider := range typedProviders {
		if k.typ.Implements(iface) {
			obj := provider()
			ResolveDeps(obj)
			out = append(out, obj)
		}
	}
	return out, true
}
