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

func NewContainer() *Containerd {
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
	key := mkey(typ)
	if ref, ok := c.objects[key]; ok {
		return ref, true
	}
	// by provider
	if provider, ok := c.providers[key]; ok {
		obj := provider()
		c.Resolve(obj)
		return obj, true
	}
	return nil, false
}

func (c *Containerd) LoadTypeObjects(iface reflect.Type) []interface{} {
	o, ok := c.LoadTypeObjectsE(iface)
	if ok {
		return o
	}
	return nil
}

func (c *Containerd) LoadTypeObjectsE(iface reflect.Type) (out []interface{}, ok bool) {
	if iface.Kind() != reflect.Interface {
		panic(fmt.Errorf("iface muse be interface type, was: %s", iface))
	}
	// objects
	for k, inst := range c.objects {
		if k.typ.Implements(iface) {
			out = append(out, inst)
		}
	}
	// providers
	for k, provider := range c.providers {
		if k.typ.Implements(iface) {
			obj := provider()
			c.Resolve(obj)
			out = append(out, obj)
		}
	}
	return out, true
}

func (c *Containerd) object(objtyp reflect.Type, obj interface{}) {
	c.objects[mkey(objtyp)] = obj
}

func (c *Containerd) provider(protyp reflect.Type, pfunc interface{}) {
	if protyp.NumOut() != 1 {
		panic(fmt.Sprintf("invalid return values of provider func, num: %d", protyp.NumOut()))
	}
	proval := reflect.ValueOf(pfunc)
	c.providers[mkey(protyp.Out(0))] = func() interface{} {
		return proval.Call(nil)[0].Interface()
	}
}

func (c *Containerd) injectSetter(meta reflect.Type, invoker reflect.Value) {
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
			if obj, ok := c.LoadObjectE(aType); ok {
				invoker.Method(i).Call([]reflect.Value{reflect.ValueOf(obj)})
			}

		case reflect.Slice:
			if aType.Elem().Kind() != reflect.Interface {
				continue
			}
			eType := aType.Elem()
			objs, ok := c.LoadTypeObjectsE(eType)
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

func mkey(typ reflect.Type) cntkey {
	return cntkey{typ: typ, name: typ.Name()}
}
