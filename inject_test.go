package runv

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type DIStructTag interface {
	Tag() string
}

type DIStructA struct {
	Name string
}

func (b *DIStructA) Tag() string {
	return "tag: a"
}

type DIStructB struct {
	RefA *DIStructA
	Name string
}

func (b *DIStructB) Tag() string {
	return "tag: b"
}

//func (b *DIStructB) SetA(a *DIStructA) {
//	fmt.Println("set: a.Name: " + a.Name)
//	b.RefA = a
//}

func (b *DIStructB) SetMultiA(a []DIStructTag) {
	for _, t := range a {
		if t != nil {
			fmt.Println("set multi: tag= " + t.Tag())
		}
	}
}

type DIStructX struct {
	NameX string
}

type DIStructY struct {
	RefX  *DIStructX
	NameY string
}

func (y *DIStructY) SetX(x *DIStructX) {
	y.RefX = x
}

type DIStructZ struct {
	RefY *DIStructY
}

func (y *DIStructZ) SetX(x *DIStructY) {
	y.RefY = x
}

func TestInjectByObject(t *testing.T) {
	cmpA := &DIStructA{Name: "DIStructA"}
	diRegisterInstance(cmpA)
	cmpB := &DIStructB{Name: "DIStructB"}
	diInjectDepens(cmpB)
	//assert.NotNil(t, cmpB.RefA)
	//assert.Equal(t, cmpB.RefA.Name, cmpA.Name)
}

func TestInjectByProvider(t *testing.T) {
	diRegisterProvider(func() *DIStructX {
		return &DIStructX{NameX: "xxxx"}
	})
	diRegisterProvider(func() *DIStructY {
		return &DIStructY{NameY: "yyyy"}
	})
	cmpZ := &DIStructZ{}
	diInjectDepens(cmpZ)
	assert.NotNil(t, cmpZ.RefY)
	assert.Equal(t, cmpZ.RefY.NameY, "yyyy")
	assert.NotNil(t, cmpZ.RefY.RefX)
	assert.Equal(t, cmpZ.RefY.RefX.NameX, "xxxx")
}
