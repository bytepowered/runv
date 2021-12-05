package runv

import (
	"fmt"
	"reflect"
	"strings"
)

type TypedKey struct {
	typ  reflect.Type
	name string
}

func (k TypedKey) String() string {
	return fmt.Sprintf("type: %s, name: %s", k.typ, k.name)
}

type Containerd struct {
	singletons map[TypedKey]interface{}        // 以Type为Key的实例列表
	factory    map[TypedKey]func() interface{} // 以Type为Key的工厂函数
	objects    []interface{}                   // 对象实例列表
	hooks      []func(*Containerd, interface{})
}

func NewContainerd() *Containerd {
	return &Containerd{
		singletons: make(map[TypedKey]interface{}, 4),
		factory:    make(map[TypedKey]func() interface{}, 4),
	}
}

func (c *Containerd) AddHook(hook func(*Containerd, interface{})) {
	c.hooks = append(c.hooks, hook)
}

func (c *Containerd) Register(in interface{}) {
	intyp := reflect.TypeOf(in)
	if intyp.Kind() == reflect.Func {
		c.factoryOf(intyp, in)
	} else {
		c.singletonOf(intyp, in)
		for _, hook := range c.hooks {
			hook(c, in)
		}
		c.Add(in)
	}
}

func (c *Containerd) Add(obj interface{}) {
	c.objects = append(c.objects, obj)
}

func (c *Containerd) Resolve(host interface{}) {
	meta := reflect.TypeOf(host)
	invoker := reflect.ValueOf(host)
	switch meta.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Struct:
		c.setter(meta, invoker)
	}
	// TODO 通过结构体字段注入
}

func (c *Containerd) LoadObject(typ reflect.Type) interface{} {
	v, ok := c.LoadObjectE(typ)
	if ok {
		return v
	}
	return nil
}

func (c *Containerd) LoadObjectE(typ reflect.Type) (interface{}, bool) {
	// singletons
	key := makeTypedKey(typ)
	if v, ok := c.singletons[key]; ok {
		return v, true
	}
	// by factory
	if fty, ok := c.factory[key]; ok {
		v := fty()
		c.Resolve(v)
		c.Add(v)
		return v, true
	}
	return nil, false
}

func (c *Containerd) LoadTyped(iface reflect.Type) []interface{} {
	o, ok := c.LoadTypedE(iface)
	if ok {
		return o
	}
	return nil
}

func (c *Containerd) LoadTypedE(iface reflect.Type) (out []interface{}, ok bool) {
	if iface.Kind() != reflect.Interface {
		panic(fmt.Errorf("arg 'iface' muse be Interface, was: %s", iface))
	}
	// singletons
	for k, obj := range c.singletons {
		if k.typ.Implements(iface) {
			out = append(out, obj)
		}
	}
	// factory
	for k, fty := range c.factory {
		if k.typ.Implements(iface) {
			v := fty()
			c.Resolve(v)
			c.Add(v)
			out = append(out, v)
		}
	}
	// objects
	for _, obj := range c.objects {
		if reflect.TypeOf(obj).Implements(iface) {
			out = append(out, obj)
		}
	}
	return out, true
}

func (c *Containerd) singletonOf(objtyp reflect.Type, obj interface{}) {
	c.singletons[makeTypedKey(objtyp)] = obj
}

func (c *Containerd) factoryOf(ftype reflect.Type, factory interface{}) {
	if ftype.NumOut() != 1 {
		panic(fmt.Sprintf("invalid return values of factory func, num: %d", ftype.NumOut()))
	}
	funcv := reflect.ValueOf(factory)
	c.factory[makeTypedKey(ftype.Out(0))] = func() interface{} {
		return funcv.Call(nil)[0].Interface()
	}
}

func (c *Containerd) setter(meta reflect.Type, invoker reflect.Value) {
	// 通过Setter函数注入
	for i := 0; i < meta.NumMethod(); i++ {
		mthtyp := meta.Method(i)
		// SetCompA(CompA), InjectComB(CompB)这样的函数
		if mthtyp.Type.NumOut() != 0 || mthtyp.Type.NumIn() != 2 {
			continue
		}
		if !strings.HasPrefix(mthtyp.Name, "Set") && !strings.HasPrefix(mthtyp.Name, "Inject") {
			continue
		}
		rettyp := mthtyp.Type.In(1)
		switch rettyp.Kind() {
		case reflect.Ptr:
			if obj, ok := c.LoadObjectE(rettyp); ok {
				invoker.Method(i).Call([]reflect.Value{reflect.ValueOf(obj)})
			}

		case reflect.Slice:
			if rettyp.Elem().Kind() != reflect.Interface {
				continue
			}
			eletyp := rettyp.Elem()
			objs, ok := c.LoadTypedE(eletyp)
			if !ok {
				continue
			}
			args := reflect.MakeSlice(reflect.SliceOf(eletyp), len(objs), len(objs))
			for at, obj := range objs {
				args.Index(at).Set(reflect.ValueOf(obj))
			}
			invoker.Method(i).Call([]reflect.Value{args})
		}
	}
}

func makeTypedKey(typ reflect.Type) TypedKey {
	return TypedKey{typ: typ, name: typ.Name()}
}
