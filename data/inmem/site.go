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
	Listeners PageListenerList
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
    pod.touched(page)
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
        pod.touched(page)
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
