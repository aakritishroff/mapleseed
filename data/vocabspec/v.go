package vocabspec

import (
	"log"
	"fmt"
)

type Key string   // might switch to int?

type Property struct {
	def string
}

type Spec struct {
	properties map[string]*Property
}

type Map struct {
	defs map[Key]string
	// 
}

// NewSpec returns a newly initialized empty Spec
func NewSpec() (vs *Spec) {
	vs = &Spec{}
	vs.properties = make(map[string]*Property)
	return
}

// NewMap returns a newly initialized empty Spec
func NewMap() (vm *Map) {
	vm = &Map{}
	vm.defs = make(map[Key]string)
	return
}

func (vs *Spec) AddProperty(name, def string) (err error) {
	vs.properties[name] = &Property{def}
	return
}

// later these need to be thread-safe...

func (vm *Map) Key(propname string, vs *Spec) (key Key, err error) {
	prop,ok := vs.properties[propname]
	if !ok {
		err = fmt.Errorf("property not defined in this vocabspec: %q", propname)
		return
	}
	for key, def := range vm.defs {
		if def == prop.def {
			return key, nil
		}
	}
	offset := 1
	key = Key(propname)
	for {
		if _,exists := vm.defs[key]; !exists {
			vm.defs[key] = prop.def
			log.Printf("added key %q for def %q", key, prop.def)
			return
		}
		offset++;
		key = Key(fmt.Sprintf("%s_%d", propname, offset))
	}
}

func (vm *Map) Prop(key Key, vs *Spec) (propname string, err error) {
	//
	return
}



