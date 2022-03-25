package runv

import (
	"fmt"
	"reflect"
	"strings"
)

type objtype struct {
	typ  reflect.Type
	name string
}

type object struct {
	typ reflect.Type
	ref interface{}
}

func (ot objtype) String() string {
	return fmt.Sprintf("type: %s, name: %s", ot.typ, ot.name)
}

type Containerd struct {
	singletons map[objtype]interface{}        // 以Type为Key的实例列表
	factory    map[objtype]func() interface{} // 以Type为Key的工厂函数
	objects    []object                       // 对象实例列表
	hooks      []func(*Containerd, interface{})
}

func NewContainerd() *Containerd {
	return &Containerd{
		singletons: make(map[objtype]interface{}, 16),
		factory:    make(map[objtype]func() interface{}, 16),
		objects:    make([]object, 0, 16),
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
	c.objects = append(c.objects, object{ref: obj, typ: reflect.TypeOf(obj)})
}

func (c *Containerd) Resolve(object interface{}) {
	meta := reflect.TypeOf(object)
	invoker := reflect.ValueOf(object)
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
	key := mkobjkey(typ)
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
	output := func(in interface{}) {
		for _, v := range out {
			if in == v {
				return
			}
		}
		out = append(out, in)
	}
	// objects
	for _, obj := range c.objects {
		if obj.typ.Implements(iface) {
			output(obj.ref)
		}
	}
	// singletons
	for k, obj := range c.singletons {
		if k.typ.Implements(iface) {
			output(obj)
		}
	}
	// factory
	for k, fty := range c.factory {
		if k.typ.Implements(iface) {
			newobj := fty()
			c.Resolve(newobj)
			c.Add(newobj)
			out = append(out, newobj)
		}
	}
	return out, true
}

func (c *Containerd) singletonOf(objtyp reflect.Type, obj interface{}) {
	c.singletons[mkobjkey(objtyp)] = obj
}

func (c *Containerd) factoryOf(ftype reflect.Type, factory interface{}) {
	if ftype.NumOut() != 1 {
		panic(fmt.Sprintf("invalid return values of factory func, num: %d", ftype.NumOut()))
	}
	funcv := reflect.ValueOf(factory)
	c.factory[mkobjkey(ftype.Out(0))] = func() interface{} {
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

func mkobjkey(typ reflect.Type) objtype {
	return objtype{typ: typ, name: typ.Name()}
}
