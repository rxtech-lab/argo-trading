package swiftargo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArray_Add(t *testing.T) {
	array := &StringArray{}

	array.Add("first")
	array.Add("second")
	array.Add("third")

	assert.Equal(t, 3, array.Size())
}

func TestArray_Get(t *testing.T) {
	array := &StringArray{}
	array.Add("first")
	array.Add("second")
	array.Add("third")

	assert.Equal(t, "first", array.Get(0))
	assert.Equal(t, "second", array.Get(1))
	assert.Equal(t, "third", array.Get(2))
}



func TestArray_Size(t *testing.T) {
	array := &StringArray{}
	assert.Equal(t, 0, array.Size())

	array.Add("first")
	assert.Equal(t, 1, array.Size())

	array.Add("second")
	array.Add("third")
	assert.Equal(t, 3, array.Size())
}

func TestArray_All(t *testing.T) {
	array := &StringArray{}
	array.Add("first")
	array.Add("second")
	array.Add("third")

	all := array.All()
	assert.Equal(t, []string{"first", "second", "third"}, all)
}

func TestArray_All_Empty(t *testing.T) {
	array := &StringArray{}

	all := array.All()
	assert.Nil(t, all)
}



func TestArray_Get_OutOfBounds(t *testing.T) {
	array := &StringArray{}
	array.Add("first")
	array.Add("second")
	array.Add("third")

	// Out of bounds returns empty string
	assert.Equal(t, "", array.Get(3))
	assert.Equal(t, "", array.Get(-1))
}