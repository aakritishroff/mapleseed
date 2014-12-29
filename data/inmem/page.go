package inmem

import (
    "sync"
	"time"
)

// When you Set one of these as the value, then its Set/Get methods
// are called to actually get/set the value.   It can't be removed.
type VirtualValue interface {
	Set(interface{}) 
	Get() interface{}
}

type Page struct {

	// public functions are threadsafe
    mutex          sync.RWMutex 

	// when a Page is added to a Site (aka Pod), this points to it,
	// and gives us the "path" for constructing the page's URL.  When
	// pod==nil, this page has no URL.
    pod            *Pod
    path           string // always starts with a slash... er NOT ANYMORE

	clusterModCount uint64

    modCount   uint64
	lastModified   time.Time

	// Pages can be delete, which is different from being removed from
	// a Site, in some subtle ways, maybe?   Like maybe we keep access control
	// when we delete a page, no know which error to give?
    deleted        bool

	// You can Add/Remove a Listener if you want to be notified when
	// this page changes.
	Listeners PageListenerList

    appData        map[string]interface{}
}

/*

The functions which modify pages (Lock) are in page_write.go

The functions which only read pages (RLock) are in page_read.go

*/
