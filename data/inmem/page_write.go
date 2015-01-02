package inmem

import (
    "encoding/json"
	// "time"
	"github.com/sandhawke/mapleseed/data/slowclock"
)


func NewPage(data ...map[string]interface{}) (page *Page, etag string) {
	page = &Page{}
    page.path = ""
    page.pod = nil
    etag = page.etag()
	page.lastModified = slowclock.Now().UTC()
	if len(data) == 0 {
		page.appData = make(map[string]interface{})
	} else if len(data) == 1 {
		page.appData = data[0]
	} else {
		panic("too many arguments to NewPage")
	}
    return
}

// This is called (via defer) at the end of every modify function to
// release the mutex and also notify anyone who needs to be notified.
// The parameter is page.modCount at the start of the function, since
// that's when defer parameters are evaluated, so we can tell if anything
// was actually changed while the lock was held.
func (page *Page) doneWithLock(startingMod uint64) {

	modified := startingMod != page.modCount

	// can we please get rid of this, soon?  Currently needed by query
	if modified && page.pod != nil && page.pod.cluster != nil {
		// shouldn't be lockr the cluster?    but might that give deadlock?
		page.clusterModCount = page.pod.cluster.getModCount()
	}

	if modified {
		// we use slowclock.Now() instead of time.Now() because
		// (1) we don't want app developers relying on this for timing, and
		// (2) using time.Now() was a huge slowdown.  This change increases
		// single-property write speed by nearly 4x in simple benchmark.
		page.lastModified = slowclock.Now().UTC()
	}

	page.mutex.Unlock()


	if modified {
		// as func so we can play with making it a goroutine
		// -- turns out wam runs ~5% slower if we do
		func () {
			page.Notify(page)
			page.Listeners.Notify(page)
			if page.pod != nil {
				//log.Printf("pod.touched")
				page.pod.touched(page)
				//log.Printf("pod.touched DONE")
			}
		}()

	}
}


// onlyIfMatch is the HTTP If-Match header value; abort if its the wrong etag
func (page *Page) SetContent(contentType string, content string, onlyIfMatch string) (etag string, notMatched bool) {
    page.mutex.Lock()
	defer page.doneWithLock(page.modCount)
    //fmt.Printf("onlyIfMatch=%q, etag=%q\n", onlyIfMatch, page.etag())
    if onlyIfMatch == "" || onlyIfMatch == page.etag() {
        if contentType == "application/json" {
            page.locked_Zero()
            page.locked_OverlayWithJSON([]byte(content))
        } else {
            page.locked_Set("contentType", contentType)
            page.locked_Set("content", content)
        }
        etag = page.etag()
    } else {
        notMatched = true
    }
    return
}

func (page *Page) SetProperties(m map[string]interface{}, onlyIfMatch string) (etag string, notMatched bool) {
    //fmt.Printf("onlyIfMatch=%q, etag=%q\n", onlyIfMatch, page.etag())
	page.mutex.Lock()
	defer page.doneWithLock(page.modCount)
	// cant use defer
    if onlyIfMatch == "" || onlyIfMatch == page.etag() {
		//log.Printf("modifying")
        page.locked_OverlayWithMap(m)
    } else {
        notMatched = true
    }
	etag = page.etag()
    return
}

// What about ACLs and what etag to use if one re-creates this URL?
// (etags need to be like "20141204-3" maybe, assuming we can remember
// deleted pages for a day.)

func (page *Page) Delete() {
    page.mutex.Lock()
    defer page.doneWithLock(page.modCount)
    page.deleted = true
	page.appData = make(map[string]interface{})
	page.modCount++
	if page.pod != nil {
		page.pod.RemovePage(page)
	}
}

func (page *Page) Undelete() {
    page.mutex.Lock()
    defer page.doneWithLock(page.modCount)
    page.deleted = false
	page.modCount++
}

func (page *Page) Set(prop string, value interface{}) {
    page.mutex.Lock()
    defer page.doneWithLock(page.modCount)
	page.locked_Set(prop, value)
	return
}

func (page *Page) locked_Set(prop string, value interface{}) {

	oldValue, exists := page.appData[prop]

	if exists {
		vv, isVirtual := oldValue.(VirtualValue)
		if isVirtual {
			oldValue = vv.Get()
			if oldValue == value {
				return
			}
			vv.Set(value)
			page.modCount++
			return
		}
	}
	if exists && oldValue == value {
		return 
	}
    if value == nil {
        delete(page.appData, prop)
    } else {
        page.appData[prop] = value
    }
	page.modCount++
	return
}


func (page *Page) locked_OverlayWithJSON(bytes []byte) {
    m := make(map[string]interface{})
    json.Unmarshal(bytes, &m)
    page.locked_OverlayWithMap(m)
}

func (page *Page) locked_OverlayWithMap(m map[string]interface{}) {
    for key, value := range m {
        page.locked_Set(key, value) 
    }
}

func (page *Page) locked_Zero() {
    for _, prop := range page.Properties() {
        if prop[0] != '_' {
            page.locked_Set(prop, nil)
        }
    }
}
