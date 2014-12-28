package inmem

import (
    //"log"
    "strings"
    "sync"
)


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

