package inmem

import (
    //"log"
    "fmt"
    "sync"
    "encoding/json"
	"time"
)

type Page struct {
    sync.RWMutex   // this library is required to be threadsafe
    pod            *Pod
    path           string // always starts with a slash
    contentType    string
    content        string
    pageModCount   uint64
	clusterModCount uint64
	lastModified   time.Time
    deleted        bool // needed for watching 404 pages, at least

    longpollQ chan chan bool   // OH, I should probably use sync.Cond instead

    /* for a !hasContent resource, the content is a JSON/RDF/whatever
    /* serialization of the appData plus some of our own metadata.
    /* For other resources, it's metadata that might be accessible
    /* somehow, eg as a nearby metadata resource. */
    // hasContent     bool
    appData        map[string]interface{}

    /* some kind of intercepter for Pod and Cluster to have their
    /* own special properties which appear when you call Get and Set,
    /* and which end up in the JSON somehow... */
    extraProperties func() (props []string)
    extraGetter    func(prop string) (value interface{}, handled bool)
    extraSetter    func(prop string, value interface{}) (handled bool)
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
}

func (page *Page) Content(accept []string) (contentType string, content string, etag string) {
    page.RLock()
    contentType = page.contentType
    content = page.content
    etag = page.etag()
    page.RUnlock()
    return
}

// onlyIfMatch is the HTTP If-Match header value; abort if its the wrong etag
func (page *Page) SetContent(contentType string, content string, onlyIfMatch string) (etag string, notMatched bool) {
    page.Lock()
    //fmt.Printf("onlyIfMatch=%q, etag=%q\n", onlyIfMatch, page.etag())
    if onlyIfMatch == "" || onlyIfMatch == page.etag() {
        if contentType == "application/json" {
            page.Zero()
            page.OverlayWithJSON([]byte(content))
        } else {
            page.contentType = contentType
            page.content = content
        }
        page.touched() // not sure if we need to keep WLock during touched()
        etag = page.etag()
    } else {
        notMatched = true
    }
    page.Unlock()
    return
}

func (page *Page) SetProperties(m map[string]interface{}, onlyIfMatch string) (etag string, notMatched bool) {
    //fmt.Printf("onlyIfMatch=%q, etag=%q\n", onlyIfMatch, page.etag())
    // we can't lock it like this, since Set also locks it... page.Lock()
    // FIXME
    if onlyIfMatch == "" || onlyIfMatch == page.etag() {
        page.OverlayWithMap(m)
        // page.touched() // not sure if we need to keep WLock during touched()
        etag = page.etag()
    } else {
        notMatched = true
    }
    // page.Unlock()
    return
}

// Delete still needs work.  For now, it just marks it as deleted and
// forgets most of the data it stores.  Actually reclaiming all
// storage would have to be done differently, since there are pointers
// to this page.  Also, what about ACLs and what etag to use if one
// re-creates this URL?  (etags need to be like "20141204-3" maybe,
// assuming we can remember deleted pages for a day.)
func (page *Page) Delete() {
    page.Lock()
    page.deleted = true
    page.contentType = ""
    page.content = ""
	page.appData = make(map[string]interface{})
    page.touched()
    page.Unlock()
}

func (page *Page) Deleted() bool {
    return page.deleted
}

/*
func (page *Page) AccessControlPage() AccessController {
}
*/

// alas, we can't just use .touched on the cluster and pod, 
// because we'd end up with infinite recursion with the page
// notifying itself


func (page *Page) touched() uint64 { // already locked
    var ch chan bool

    page.pod.podTouched()
    page.pageModCount++
	page.clusterModCount = page.pod.cluster.modCount
	page.lastModified = time.Now().UTC()

	// switch to cond var?
    for {
        select {
        case ch = <-page.longpollQ:
            ch <- true // let them know they can go on!
        default:
            return page.pageModCount
        }
    }
}
func (page *Page) WaitForNoneMatch(etag string) {
    page.RLock()
    // don't use defer, since we need to unlock before done
    if etag != page.etag() {
        page.RUnlock()
        return
    }
    ch := make(chan bool)
    page.longpollQ <- ch // queue up ch as a response point for us
    page.RUnlock()
    _ = <-ch // wait for that response
}



//////////////////////////////

// maybe we should generalize these as virtual properties and
// have a map of them?   But some of them are the same for every page...
//
// should we have _contentType and _content among them?
//
// that would let us serialize nonData resources in json

func (page *Page) Properties() (result []string) {
    result = make([]string,0,len(page.appData)+5)
    result = append(result, "_id")
    result = append(result, "_etag")
    result = append(result, "_owner")
    result = append(result, "_lastModified")
    if page.contentType != "" {
        result = append(result, "_contentType")
        result = append(result, "_content")
    }
    if page.extraProperties != nil {
        result = append(result, page.extraProperties()...)
    }
    page.RLock()
    for k := range page.appData {
        result = append(result, k)
    }
    page.RUnlock()
    return
}

func (page *Page) GetDefault(prop string, def interface{}) (value interface{}) {
    value, exists := page.Get(prop)
    if !exists { value = def }
    return
}

func (page *Page) Get(prop string) (value interface{}, exists bool) {
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
    if prop == "_contentType" {
        return page.contentType, true
    }
    if prop == "_content" {
        return page.content, true
    }
    if page.extraGetter != nil {
        value, exists = page.extraGetter(prop)
        if exists {
            return value, true
        }
    }
    page.RLock()
    value, exists = page.appData[prop]
    page.RUnlock()
    return
}

func (page *Page) Set(prop string, value interface{}) {
    if page.extraSetter != nil {
        handled := page.extraSetter(prop, value)
        if handled { return }
    }
	// Why do we special case these...?
    if prop == "_contentType" {
        page.contentType = value.(string)
        return
    }
    if prop == "_content" {
        page.content = value.(string)
        return
    }
    if prop[0] == '_' || prop[0] == '@' {
        // don't allow any (other) values to be set like this; they
        // are ours to handle.   We COULD give an error...?
        return
    }
    page.Lock()
    if value == nil {
        delete(page.appData, prop)
    } else {
        if page.appData == nil { page.appData = make(map[string]interface{})}
        page.appData[prop] = value
    }
    page.Unlock()
    page.touched();
    return
}

func (page *Page) MarshalJSON() (bytes []byte, err error) {
    return json.Marshal(page.AsJSON)
}

func (page *Page) AsJSON() map[string]interface{} {
    // extra copying for simplicity for now
    props := page.Properties() 
    m := make(map[string]interface{}, len(props))
    page.RLock()
    for _, prop := range props {
        value, handled := page.Get(prop)
        if handled { m[prop] = value }
    }
    page.RUnlock()
    //fmt.Printf("Going to marshal %q for page %q, props %q", m, page, props)
    return m
    //return []byte(""), nil
}

func (page *Page) OverlayWithJSON(bytes []byte) {
    m := make(map[string]interface{})
    json.Unmarshal(bytes, &m)
    page.OverlayWithMap(m)
}

func (page *Page) OverlayWithMap(m map[string]interface{}) {
    for key, value := range m {
        page.Set(key, value)   // do we need to recurse...?
    }
}

func (page *Page) Zero() {
    for _, prop := range page.Properties() {
        if prop[0] != '_' {
            page.Set(prop, nil)
        }
    }
}
