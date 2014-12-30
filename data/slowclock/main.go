package slowclock

/* 
    use slowclock.Now() instead of time.Now() at those times
    when time.Now() is too expensive.
*/

import (
	"time"
	"sync"
)

func init() {
	go updater()
}

var now time.Time
var lock sync.Mutex

func updater() {
	for {
		t := time.Now()
		t = t.Truncate(100*time.Millisecond)
		//fmt.Printf("time now %q\n", now)
		lock.Lock()
		now = t
		lock.Unlock()
		time.Sleep(100*time.Millisecond)

	}
}

func Now() (result time.Time) {
	lock.Lock()
	result = now
	lock.Unlock()
	return
}


