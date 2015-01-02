package inmem

import "log"

var Trace bool

func trace(template string, args ...interface{}) {
	if Trace {
		log.Printf("inmem."+template, args...)
	}
}

