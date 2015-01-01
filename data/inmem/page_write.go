package inmem

import (
    "encoding/json"
	"time"
	"../slowclock"
)


func NewPage() (page *Page, etag string) {
	page = &Page{}
    page.path = ""
    page.pod = nil
    etag = page.etag()
	page.lastModified = time.Now().UTC()
	page.appData = make(map[string]interface{})
    return
}

// This is called (via defer) at the end of every modify function to
// release the mutex and also notify anyone who needs to be notified.
// The parameter is page.modCount at the start of the function, since
// that's when defer parameters are evaluated, so we can tell if anything
// was actually changed while the lock was held.
func (page *Page) doneWithLock(startingMod uint64) {

	modified := startingMod != page.modCount

	page.mutex.Unlock()

	if modified {
		// we use slowclock.Now() instead of time.Now() because
		// (1) we don't want app developers relying on this for timing, and
		// (2) using time.Now() was a huge slowdown.  This change increases
		// single-property write speed by nearly 4x in simple benchmark.
		page.lastModified = slowclock.Now().UTC()

		page.Listeners.Notify(page);
		if page.pod != nil {
			page.pod.touched(page)
			if page.pod.cluster != nil {
				// I'd like to remove this hack soon....  needed by
				// current query infrastructure
				page.clusterModCount = page.pod.cluster.modCount
			}
		}
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

// Delete still needs work.  For now, it just marks it as deleted and
// forgets most of the data it stores.  Actually reclaiming all
// storage would have to be done differently, since there are pointers
// to this page.  Also, what about ACLs and what etag to use if one
// re-creates this URL?  (etags need to be like "20141204-3" maybe,
// assuming we can remember deleted pages for a day.)
func (page *Page) Delete() {
    page.mutex.Lock()
    defer page.doneWithLock(page.modCount)
    page.deleted = true
	page.appData = make(map[string]interface{})
	page.modCount++
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