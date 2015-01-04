package abstract

import "log"

// easy way to turn tracing on/off for the whole package

func trace(template string, args ...interface{}) {
	if false {
		log.Printf("data/abstract: "+template, args...)
	}
}

