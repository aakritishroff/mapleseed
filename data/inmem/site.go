/*

   please rename this to Site...?

*/


package inmem

import (
    "fmt"
    "sync"
	"strings"
	"golang.org/x/crypto/bcrypt"
)



type Pod struct {
	notifier
    sync.RWMutex  
    rootPage      *Page   // public profile info
	configPage    *Page   // private profile info
    urlWithSlash  string  // just to be clear pod urls MUST end in slash
    cluster       *Cluster
    pages         map[string]*Page
    newPageNumber uint64  // ?switch to these being suffix for particular prefix
	Listeners     PageListenerList  // ?switch to private
	fullyLoaded   bool  // no pages still on disk
	pwHash        []byte
}


func NewPod(url string) (pod *Pod) {
    pod = &Pod{}
	pod.fullyLoaded = true
	if !strings.HasSuffix(url, "/") {
		// or should we flag an error?   eh, this seems okay.
		url = url+"/"
	}
	pod.urlWithSlash = url
    pod.pages = make(map[string]*Page)
	pod.rootPage,_ = pod.PageByPath("", true)
	pod.rootPage.Set("_isPod", true)
	// ...
	return
}

func (pod *Pod) SetPassword(newPassword string) {
	cost := 10   // should really set it to whatever takes 0.01s
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), cost)
    if err != nil {
        panic(err)
    }
	pod.pwHash = hashedPassword
	pod.save()
}


func (pod *Pod) HasPassword(password string) bool {
    err := bcrypt.CompareHashAndPassword(pod.pwHash, []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false
	}
	if err == nil {
		return true
	}
	panic(err)
}

func (pod *Pod) touched(page *Page) {

	pod.Notify(page)

	pod.Listeners.Notify(page);

    if pod.cluster != nil {
		pod.cluster.clusterTouched(page)
	}

}

func (pod *Pod) URL() string {
    return pod.urlWithSlash
}


func (pod *Pod) Pages() (result []*Page) {
	pod.loadAllPages()
	return pod.loadedPages()
}

func (pod *Pod) loadedPages() (result []*Page) {
    pod.RLock()
    result = make([]*Page, 0, len(pod.pages))
    for _, k := range pod.pages {
        result = append(result, k)
    }
    pod.RUnlock()
    return
}




/*  
   It's tempting to switch to having folks create a page, then
   add it to the site, but it gets complicated dealing with assigning
   a new unique path, or if the page already has a path that's already
   taken.   So we'll leave it like this for now.
*/

func (pod *Pod) uniquePath() (path string) {
    for {
        path = fmt.Sprintf("a%d", pod.newPageNumber)
        pod.newPageNumber++
        if _, taken := pod.pages[path]; taken {
			continue
        }
		if !pod.fullyLoaded {
			if pod.pathTakenOnDisk(path) {
				continue
			}
		}
		break
    }
	return
}

func (pod *Pod) NewPage(data ...map[string]interface{}) (page *Page, etag string) {
	page,_ = NewPage(data...)
    pod.Lock()
    var path string
    page.path = pod.uniquePath()
    page.pod = pod
    etag = page.etag()
    pod.pages[path] = page

	pod.Unlock()

	pod.touched(page)
    return
}

func (pod *Pod) RemovePage(page *Page) {
    pod.Lock()
	defer pod.Unlock()
	delete(pod.pages, page.path)
}


func (pod *Pod) PageByPath(path string, mayCreate bool) (page *Page, created bool) {
    pod.Lock()
	defer pod.Unlock()

	//log.Printf("pagebypath: %s", path);

    page, _ = pod.pages[path]
	if !pod.fullyLoaded && page == nil {
		page,_ = NewPage()
		page.path = path
		page.pod = pod
		//log.Printf("pagebypath trying load: %s", path);
		loaded, err := page.load()
		if loaded {
			pod.pages[path] = page
			return
		} 
		if err != nil {
			panic(err)
		}
	}
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


// so that we can access this via an interface
func (pod *Pod) AddListener(l chan interface{}) {
	pod.Listeners.Add(l)
}
