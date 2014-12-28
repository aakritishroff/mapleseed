package inmem

import (
    //"log"
    "fmt"
    "strings"
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

func (cluster *Cluster) clusterTouched() {
    cluster.modlock.Lock()
    cluster.modCount++
    cluster.modlock.Unlock()
    cluster.modified.Broadcast()
}
func (cluster *Cluster) WaitForModSince(ver uint64) {
    //log.Printf("WaitForModSince %d %d 1", ver, cluster.modCount);
    // this should work with RLock, but it doesn't
    cluster.modlock.Lock()
    //log.Printf("WaitForModSince %d %d 2", ver, cluster.modCount);
    // NO defer, since we need to unlock before done

    if ver == cluster.modCount {
        //log.Printf("WaitForModSince %d %d 3", ver, cluster.modCount);
        cluster.modified.Wait() // internally does Unlock()
        //log.Printf("WaitForModSince %d %d 4", ver, cluster.modCount);
        // despite the badly worded sync.Cond documentation, 
        // Wait() returns with modlock held.
    }
    cluster.modlock.Unlock()
    //log.Printf("WaitForModSince %d %d 5", ver, cluster.modCount);
}


func (pod *Pod) podTouched() {
    pod.cluster.clusterTouched()
}

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

type Pod struct {
    sync.RWMutex  
    rootPage      *Page
    urlWithSlash  string  // just to be clear pod urls MUST end in slash
    cluster       *Cluster
    pages         map[string]*Page
    newPageNumber uint64
}

func (pod *Pod) URL() string {
    return pod.urlWithSlash
}

//func (pod *Pod) Pages() Selection {
//}

func (pod *Pod) Pages() (result []*Page) {
    pod.RLock()
    result = make([]*Page, 0, len(pod.pages))
    for _, k := range pod.pages {
        result = append(result, k)
    }
    pod.RUnlock()
    return
}

func (pod *Pod) NewPage() (page *Page, etag string) {
    pod.Lock()
    var path string
    for {
        path = fmt.Sprintf("a%d", pod.newPageNumber)
        pod.newPageNumber++
        if _, taken := pod.pages[path]; !taken {
            break
        }
    }
    page = &Page{}
    page.path = path
    page.pod = pod
    etag = page.etag()
    pod.pages[path] = page
    pod.podTouched()
    pod.Unlock()
    return
}
func (pod *Pod) PageByPath(path string, mayCreate bool) (page *Page, created bool) {
    pod.Lock()
	defer pod.Unlock()

	// fmt.Printf("pagebypath: %s", path);

    page, _ = pod.pages[path]
    if mayCreate && page == nil {
        page = &Page{}
        page.path = path
        page.pod = pod
        pod.pages[path] = page
        created = true
        pod.podTouched()
		return
    }
    if mayCreate && page.deleted {
        page.deleted = false
        created = true
        // trusting you'll set the content, and that'll trigger a TOUCHED
		return
    }
	if !mayCreate && page != nil && page.deleted {
		page = nil
		return
	}
	return
}

func (pod *Pod) PageByURL(url string, mayCreate bool) (page *Page, created bool) {
	if len(url) < len(pod.urlWithSlash) {
		return nil, false
	}
    path := url[len(pod.urlWithSlash):]
    return pod.PageByPath(path, mayCreate)
}

type Cluster struct {
    Page
	PodURLTemplate string
	HubURL string
    url  string // which should be the same as URL(), but that's recursive
    pods map[string]*Pod
    modCount uint64
    modlock sync.RWMutex   // just used to lock modCount
    modified *sync.Cond
}

func (cluster *Cluster) ModCount() uint64 {
    // FIXME in theory should do a RLock, in case the increment is not atomic
    cluster.modlock.RLock()
    defer cluster.modlock.RUnlock()
    return cluster.modCount
}

// The URL is the nominal URL of the cluster itself.  It does
// not have to be syntactically related to its pod URLs
func NewInMemoryCluster(url string) (cluster *Cluster) {
    cluster = &Cluster{}
    cluster.url = url
    cluster.pods = make(map[string]*Pod)
    cluster.modified = sync.NewCond(&cluster.modlock)

    // and as a page?
    // leave that stuff zero for now
    return
}

func (cluster *Cluster) Pods() (result []*Pod) {
    cluster.RLock()
    defer cluster.RUnlock()
    result = make([]*Pod, 0, len(cluster.pods))
    for _, k := range cluster.pods {
        result = append(result, k)
    }
    return
}

func (cluster *Cluster) NewPod(url string) (pod *Pod, existed bool) {
    //  !!!!!!!    commenting out only to test soemthing.
    //cluster.Lock()
    //defer cluster.Unlock()

    if pod, existed = cluster.pods[url]; existed {
        return
    }
    pod = &Pod{}
    pod.cluster = cluster
	if !strings.HasSuffix(url, "/") {
		// or should we flag an error?   eh, this seems okay.
		url = url+"/"
	}
	pod.urlWithSlash = url
    pod.pages = make(map[string]*Page)
    cluster.pods[url] = pod
    existed = false
	pod.rootPage,_ = pod.PageByPath("", true)
	pod.rootPage.Set("_isPod", true)
	// fill in more about the user....?
    cluster.clusterTouched()
    return
}

func (cluster *Cluster) PodByURL(url string) (pod *Pod) {
    cluster.RLock()
    pod = cluster.pods[url]
    cluster.RUnlock()
    return
}

func (cluster *Cluster) PageByURL(url string, mayCreate bool) (page *Page, created bool) {
    // if we had a lot of pods we could hardcode some logic about
    // what their URLs look like, but for now this should be fine.
    cluster.RLock()
    defer cluster.RUnlock()
    for _, pod := range cluster.pods {
        if strings.HasPrefix(url, pod.urlWithSlash) {
            page, created = pod.PageByURL(url, mayCreate)
            return
        }
    }
    return
}

type PageFilter func (page *Page) bool

func (cluster *Cluster) CollectMatchingPages(filter PageFilter, results *[]*Page) {
    cluster.RLock()
    defer cluster.RUnlock()
    for _, pod := range cluster.pods {
        pod.CollectMatchingPages(filter, results)
    }
}
func (pod *Pod) CollectMatchingPages(filter PageFilter, results *[]*Page) {
    pod.RLock()
    defer pod.RUnlock()
    for _, page := range pod.pages {
        page.CollectMatchingPages(filter, results)
    }
}
func (page *Page) CollectMatchingPages(filter PageFilter, results *[]*Page) {
    page.RLock()
    defer page.RUnlock()
    if filter(page) {
        *results = append(*results, page)
    }
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

// as per http://golang.org/pkg/sort/ example
type ByURL []*Page
func (a ByURL) Len() int           { return len(a) }
func (a ByURL) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByURL) Less(i, j int) bool { return a[i].URL() < a[j].URL() }

// to be more flexible we have to wrap them...
// http://play.golang.org/p/4PmJVi2_7D
