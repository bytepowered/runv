package runv

import (
	"fmt"
	"reflect"
	"strings"
)

type cntkey struct {
	typ  reflect.Type
	name string
}

func (k cntkey) String() string {
	return fmt.Sprintf("type: %s, name: %s", k.typ, k.name)
}

type Containerd struct {
	objects   map[cntkey]interface{}
	providers map[cntkey]func() interface{}
	hooks     []func(*Containerd, interface{})
}

func NewContainerd() *Containerd {
	return &Containerd{
		objects:   make(map[cntkey]interface{}, 4),
		providers: make(map[cntkey]func() interface{}, 4),
	}
}

func (c *Containerd) AddHook(hook func(*Containerd, interface{})) {
	c.hooks = append(c.hooks, hook)
}

func (c *Containerd) Register(in interface{}) {
	intyp := reflect.TypeOf(in)
	if intyp.Kind() == reflect.Func {
		c.provider(intyp, in)
	} else {
		c.object(intyp, in)
		for _, hook := range c.hooks {
			hook(c, in)
		}
	}
}

func (c *Containerd) Resolve(host interface{}) {
	meta := reflect.TypeOf(host)
	invoker := reflect.ValueOf(host)
	switch meta.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Struct:
		c.injectSetter(meta, invoker)
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
	// instances
	key := mkcntkey(typ)
	if ref, ok := c.objects[key]; ok {
		return ref, true
	}
	// by provider
	if profun, ok := c.providers[key]; ok {
		obj := profun()
		c.Resolve(obj)
		return obj, true
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

func (c *Containerd) LoadTypedE(iface reflect.Type) (objs []interface{}, ok bool) {
	if iface.Kind() != reflect.Interface {
		panic(fmt.Errorf("arg 'iface' muse be Interface, was: %s", iface))
	}
	// objects
	for k, obj := range c.objects {
		if k.typ.Implements(iface) {
			objs = append(objs, obj)
		}
	}
	// providers
	for k, profun := range c.providers {
		if k.typ.Implements(iface) {
			obj := profun()
			c.Resolve(obj)
			objs = append(objs, obj)
		}
	}
	return objs, true
}

func (c *Containerd) object(objtyp reflect.Type, obj interface{}) {
	c.objects[mkcntkey(objtyp)] = obj
}

func (c *Containerd) provider(protyp reflect.Type, pfunc interface{}) {
	if protyp.NumOut() != 1 {
		panic(fmt.Sprintf("invalid return values of provider func, num: %d", protyp.NumOut()))
	}
	proval := reflect.ValueOf(pfunc)
	c.providers[mkcntkey(protyp.Out(0))] = func() interface{} {
		return proval.Call(nil)[0].Interface()
	}
}

func (c *Containerd) injectSetter(meta reflect.Type, invoker reflect.Value) {
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

func mkcntkey(typ reflect.Type) cntkey {
	return cntkey{typ: typ, name: typ.Name()}
}
