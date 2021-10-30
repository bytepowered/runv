package runv

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type DIStructA struct {
	Name string
}

type DIStructB struct {
	RefA *DIStructA
	Name string
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
	cmpA := &DIStructA{Name: "abcdef"}
	diRegisterObject(cmpA)
	cmpB := &DIStructB{Name: "DIStructB"}
	diInjectDepens(cmpB)
	assert.NotNil(t, cmpB.RefA)
	assert.Equal(t, cmpB.RefA.Name, cmpA.Name)
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
