package vocabspec

import (
	"testing"
	"log"
)

func TestSameDefSameKey(t *testing.T) {

	def := "The name by which the person is primarily called"
	vs1 := NewSpec()
	vs1.AddProperty("name", def)

	vs2 := NewSpec()
	vs2.AddProperty("fullname", def)

	vm := NewMap()

	key1,err := vm.Key("name", vs1)
	if err != nil {
		t.Error(err)
	}
	key2,err := vm.Key("fullname", vs2)
	if err != nil {
		t.Error(err)
	}
	if key1 != key2 {
		t.Error("key1 != key2", key1, key2)
	}
	log.Printf("WORKED!  %q == %q", key1, key2)
}
