package slowclock

import (
	"testing"
	"log"
	"time"
)

func manualTest1(t *testing.T) {
	for {
		log.Printf("%q %q", Now(), time.Now())
		time.Sleep(100*time.Millisecond)
	}
}
