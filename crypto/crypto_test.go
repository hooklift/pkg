package crypto

import (
	"testing"

	"github.com/hooklift/assert"
)

func TestEncription(t *testing.T) {
	input := []byte("hola mundo")
	key := NewKey()
	e, ok := Encrypt(key, input)
	assert.Cond(t, ok == true, "got: %v, expected: %v", ok, true)
	assert.Cond(t, e != nil, "got: %v, expected: %v", e, nil)

	d, ok := Decrypt(key, e)
	assert.Cond(t, ok == true, "got: %v, expected: %v", ok, true)
	assert.Equals(t, input, d)
	assert.Equals(t, string(input[:]), string(d[:]))
}

func TestHashPassword(t *testing.T) {
	p := "hola mundo"
	salt := "test"
	h := HashPassword(p, salt)

	assert.Cond(t, h != "", "hashed value should not be empty")

	// It should produce the same hash every time given the same input string
	h2 := HashPassword(p, salt)
	assert.Equals(t, h, h2)
}

