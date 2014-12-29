/*

   please rename this to Site...?

*/


package inmem

import (
    //"log"
    "fmt"
    "sync"
)



type Pod struct {
    sync.RWMutex  
    rootPage      *Page
    urlWithSlash  string  // just to be clear pod urls MUST end in slash
    cluster       *Cluster
    pages         map[string]*Page
    newPageNumber uint64
	Listeners     PageListenerList
}


func (pod *Pod) touched(page *Page) {
	pod.Listeners.Notify(page);
    if pod.cluster != nil {
		pod.cluster.clusterTouched()
	}
}

func (pod *Pod) URL() string {
    return pod.urlWithSlash
}


func (pod *Pod) Pages() (result []*Page) {
    pod.RLock()
    result = make([]*Page, 0, len(pod.pages))
    for _, k := range pod.pages {
        result = append(result, k)
    }
    pod.RUnlock()
    return
}

/* 

Do we want

  Add
  Remove

of existing pages?     *shrug*

*/


func (pod *Pod) NewPage() (page *Page, etag string) {
	page,_ = NewPage()
    pod.Lock()
    var path string
    for {
        path = fmt.Sprintf("a%d", pod.newPageNumber)
        pod.newPageNumber++
        if _, taken := pod.pages[path]; !taken {
            break
        }
    }
    page.path = path
    page.pod = pod
    etag = page.etag()
    pod.pages[path] = page
    pod.Unlock()
    return
}
func (pod *Pod) PageByPath(path string, mayCreate bool) (page *Page, created bool) {
    pod.Lock()
	defer pod.Unlock()

	// fmt.Printf("pagebypath: %s", path);

    page, _ = pod.pages[path]
    if mayCreate && page == nil {
		page,_ = NewPage()
        page.path = path
        page.pod = pod
        pod.pages[path] = page
        created = true
		return
    }
    if mayCreate && page.deleted {
		// tiny race condition between this check and undelete...
        page.Undelete()
        created = true
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
