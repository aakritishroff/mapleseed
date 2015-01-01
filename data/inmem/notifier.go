package inmem

/*
   Like "listener", but we use callbacks instead of channels, it's not
   Page-specific, and it's a mix-in.

*/

import ( 
	"sync"
	//"log"
)

type Callback *func(interface{})

type notifier struct {
	mutex sync.RWMutex
	callbacks []Callback;
}

func (n *notifier) AddCallback(cb *func(interface{})) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.callbacks = append(n.callbacks, cb);
}

// remove any occurances of cb from this set of callbacks
func (n *notifier) RemoveCallback(cb Callback) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	new := make([]Callback, len(n.callbacks)-1)
	for _,value := range n.callbacks {
		if value != cb {
			new = append(new, value);
		}
	}
	n.callbacks = new
}

func (n *notifier) Notify(data interface{}) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	for _,cb := range n.callbacks {
		//log.Printf("Notify cb %d", i)
		(*cb)(data)
	}
}
