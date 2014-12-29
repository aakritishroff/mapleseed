package inmem

import (
    "log"
    "fmt"
    "sync"
    "encoding/json"
	"time"
)

/*
// We could do something like this, to allow for the values
// inside app-data to be properly typed?  Or we could use 
// reflection (look at how json library does it).  We'd like
// to be able to have a page pointer be the pointer when seen
// internally, and the URL when seen externally....
type Value interface {
	AsString() (value string, err error)
	AsFloat64() (value float64, err error)
	AsTime() (value time.Time, err error)
	AsPage() (value *Page, err error)
}
*/

type ValueManager interface {
	Set(interface{}) 
	Get() interface{}
}

// NO, do this:
type VirtualValue interface {
	Set(interface{}) 
	Get() interface{}
}
// when this is SET as the value, then after that
// gets and sets RUN this, instead of actually
// changing the value.    Can it remove itself...?


type Page struct {
    mutex          sync.RWMutex // public functions are threadsafe
    pod            *Pod
    path           string // always starts with a slash... er NOT ANYMORE
    pageModCount   uint64
	clusterModCount uint64
	lastModified   time.Time
    deleted        bool // needed for watching 404 pages, at least

	Listeners PageListenerList
    // longpollQ chan chan bool   // OH, I should probably use sync.Cond instead

    /* for a !hasContent resource, the content is a JSON/RDF/whatever
    /* serialization of the appData plus some of our own metadata.
    /* For other resources, it's metadata that might be accessible
    /* somehow, eg as a nearby metadata resource. */
    // hasContent     bool
    appData        map[string]interface{}

	virtualAppData map[string]ValueManager
}

func NewPage() (page *Page, etag string) {
	page = &Page{}
    page.path = ""
    page.pod = nil
    etag = page.etag()
	page.lastModified = time.Now().UTC()
	page.appData = make(map[string]interface{})
    return
}

func (page *Page) Pod() *Pod {
    return page.pod
}

// this is private because no one should be getting an etag separate from
// getting or setting content.   if they did, they'd likely have a race
// condition
func (page *Page) etag() string {
    return fmt.Sprintf("%d", page.pageModCount)
}
func (page *Page) LastModifiedAtClusterModCount() uint64 {
    return page.clusterModCount
}
func (page *Page) Path() string {
    return page.path
}
func (page *Page) URL() string {
    return page.pod.urlWithSlash + page.path
	//   wait, what?   that's two slashes!
}


func (page *Page) Content(accept []string) (contentType string, content string, etag string) {
    page.mutex.RLock()
	defer page.mutex.RUnlock()
    ct, typeExists := page.Get("contentType")
    c, contentExists := page.Get("content")
    etag = page.etag()
	if typeExists && contentExists {
		contentType = ct.(string)
		content = c.(string)
		return
	}
	if contentExists {
		// ?! not sure how to handle this  
		contentType = "text/plain"
		return
	}
	bytes, err := page.MarshalJSON()
	if err != nil {
		log.Printf("cant marshal json: %s", err)
		contentType = "text/plain"
		content = ""
	} else {
		contentType = "application/json"
		content = string(bytes)
	}
	return
}

// onlyIfMatch is the HTTP If-Match header value; abort if its the wrong etag
func (page *Page) SetContent(contentType string, content string, onlyIfMatch string) (etag string, notMatched bool) {
	modified := false
    page.mutex.Lock()
	defer page.doneWithLock(&modified)
    //fmt.Printf("onlyIfMatch=%q, etag=%q\n", onlyIfMatch, page.etag())
    if onlyIfMatch == "" || onlyIfMatch == page.etag() {
        if contentType == "application/json" {
            page.locked_Zero()
            page.locked_OverlayWithJSON([]byte(content))
        } else {
            page.locked_Set("contentType", contentType)
            page.locked_Set("content", content)
        }
		modified = true
        etag = page.etag()
    } else {
        notMatched = true
    }
    return
}

func (page *Page) SetProperties(m map[string]interface{}, onlyIfMatch string) (etag string, notMatched bool) {
    //fmt.Printf("onlyIfMatch=%q, etag=%q\n", onlyIfMatch, page.etag())
	modified := false
	page.mutex.Lock()
	// cant use defer
    if onlyIfMatch == "" || onlyIfMatch == page.etag() {
		//log.Printf("modifying")
        page.locked_OverlayWithMap(m)
		modified = true  // TODO check if it was really modified
    } else {
        notMatched = true
    }
	page.doneWithLock(&modified)
	etag = page.etag()  // needs to be AFTER doneWithLock runs (so no defer)
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
    page.deleted = true
	page.appData = make(map[string]interface{})
	modified := true
    page.doneWithLock(&modified)
}

func (page *Page) Deleted() bool {
    return page.deleted
}

func (page *Page) doneWithLock(modified *bool) {

	//log.Printf("doneWithLock(%q)", modified)
	if *modified {

		//log.Printf(".. modified")
		page.pageModCount++
		//log.Printf(".. counter %q", page.pageModCount)
		page.lastModified = time.Now().UTC()

		// this doesn't work -- we can't safely update the cluster modcount
		if page.pod != nil {
			page.clusterModCount = page.pod.cluster.modCount
		}

	} else {
		//log.Printf(".. not modified")
	}
	page.mutex.Unlock()

	page.Listeners.Notify(page);
    if page.pod != nil {
		page.pod.touched(page)
	}
}

func (page *Page) WaitForNoneMatch(etag string) {
    page.mutex.RLock()
    // don't use defer, since we need to unlock before done
    if etag != page.etag() {
        page.mutex.RUnlock()
        return
    }
    ch := make(chan *Page)
	page.Listeners.Add(ch)
	page.mutex.RUnlock()
	_ = <- ch
	page.Listeners.Remove(ch)
}



func (page *Page) Properties() (result []string) {
    result = make([]string,0,len(page.appData)+4)

	// should we even bother to include these two, since they're so obvious?
    result = append(result, "_id")     
    result = append(result, "_owner")

    result = append(result, "_etag")
    result = append(result, "_lastModified")

    page.mutex.RLock()
    for k := range page.appData {
        result = append(result, k)
    }
    for k := range page.virtualAppData {
        result = append(result, k)
    }
    page.mutex.RUnlock()
    return
}

func (page *Page) GetDefault(prop string, def interface{}) (value interface{}) {
    value, exists := page.Get(prop)
    if !exists { value = def }
    return
}

func (page *Page) Get(prop string) (value interface{}, exists bool) {
    page.mutex.RLock()
    defer page.mutex.RUnlock()
	return page.locked_Get(prop)
}

func (page *Page) locked_Get(prop string) (value interface{}, exists bool) {
    if prop == "_id" {
        if page.pod == nil { return "", false }
        return page.URL(), true
    }
    if prop == "_etag" {  // please lock first, if using this!
        return page.etag(), true
    }
    if prop == "_owner" { 
        //if page.pod == nil { return interface{}(page).(*Cluster).url, true }
        if page.pod == nil { return "", false }
        return page.pod.urlWithSlash, true
    }
    if prop == "_lastModified" {
        return page.lastModified.Format(time.RFC3339Nano), true
    }
    value, exists = page.appData[prop]
	if ! exists {
		manager, exists := page.virtualAppData[prop]
		if exists {
			value = manager.Get()
		}
	}
    return
}

func (page *Page) Set(prop string, value interface{}) {
    page.mutex.Lock()
	modified := false
	defer page.doneWithLock(&modified)
	oldValue, exists := page.locked_Get(prop)
	if exists && oldValue == value {
		log.Printf("page.Set(%q,%q) doesn't change anything (was %q)", prop, value, oldValue);
		return
	}
	modified = true
	page.locked_Set(prop, value)
	//log.Printf("modified=%q", modified)
}

func (page *Page) locked_Set(prop string, value interface{}) {

	manager, managerExists := page.virtualAppData[prop]
	if managerExists {
		manager.Set(value)
		return
	} 

    if value == nil {
        delete(page.appData, prop)
    } else {
        if page.appData == nil { 
			page.appData = make(map[string]interface{})
		}
        page.appData[prop] = value
    }

    return
}

func (page *Page) MarshalJSON() (bytes []byte, err error) {
    return json.Marshal(page.AsJSON)
}

// Return a JSON-able map[] of all the data in the page
func (page *Page) AsJSON() map[string]interface{} {
    // extra copying for simplicity for now
    page.mutex.RLock()
    props := page.Properties() 
    m := make(map[string]interface{}, len(props))
    for _, prop := range props {
        value, handled := page.Get(prop)
        if handled { m[prop] = value }
    }
    page.mutex.RUnlock()
    //fmt.Printf("Going to marshal %q for page %q, props %q", m, page, props)
    return m
    //return []byte(""), nil
}

func (page *Page) locked_OverlayWithJSON(bytes []byte) {
    m := make(map[string]interface{})
    json.Unmarshal(bytes, &m)
    page.locked_OverlayWithMap(m)
}

func (page *Page) locked_OverlayWithMap(m map[string]interface{}) {
    for key, value := range m {
        page.locked_Set(key, value)   // do we need to recurse...?
    }
}

func (page *Page) locked_Zero() {
    for _, prop := range page.Properties() {
        if prop[0] != '_' {
            page.locked_Set(prop, nil)
        }
    }
}
