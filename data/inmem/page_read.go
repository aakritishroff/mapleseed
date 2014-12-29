package inmem

import (
    "log"
    "fmt"
    "encoding/json"
	"time"
)

func (page *Page) Pod() *Pod {
    return page.pod
}

// this is private because no one should be getting an etag separate from
// getting or setting content.   if they did, they'd likely have a race
// condition
func (page *Page) etag() string {
    return fmt.Sprintf("%d", page.modCount)
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

func (page *Page) Deleted() bool {
    return page.deleted
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
    for k,value := range page.appData {
		vv, isVirtual := value.(VirtualValue)
		if isVirtual {
			value = vv.Get()
			if value == nil {
				continue
			}
		}
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

// WHY do we need to return "exists", when value==nil means it doesn't?
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
	vv, isVirtual := value.(VirtualValue)
	if isVirtual {
		value = vv.Get()
		if value == nil {
			exists = false
		}
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

