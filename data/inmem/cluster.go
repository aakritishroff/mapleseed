package inmem

// call this "webview" maybe?

import (
	"errors"
	"log"
	"strings"
	"sync"
)

type Cluster struct {
	notifier
	mutex sync.RWMutex // public functions are threadsafe
	pods  map[string]*Pod

	Listeners PageListenerList

	PodURLTemplate string
	HubURL         string
	url            string // which should be the same as URL(), but that's recursive
	modCount       uint64
	modlock        sync.RWMutex // just used to lock modCount
	modified       *sync.Cond
	fsroot         string
	queueForFSB    Listener
}

func (cluster *Cluster) clusterTouched(page *Page) {
	cluster.modlock.Lock()
	cluster.modCount++
	cluster.modlock.Unlock()
	cluster.modified.Broadcast()
	cluster.Notify(page)
	cluster.Listeners.Notify(page)
}

func (cluster *Cluster) getModCount() uint64 {
	cluster.modlock.Lock()
	defer cluster.modlock.Unlock()
	return cluster.modCount
}

func (cluster *Cluster) WaitForModSince(ver uint64) {
	//log.Printf("WaitForModSince %d %d 1", ver, cluster.modCount);
	// this should work with rlock, but it doesn't
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

func (cluster *Cluster) ModCount() uint64 {
	// FIXME in theory should do a rlock, in case the increment is not atomic
	cluster.modlock.Lock()
	defer cluster.modlock.Unlock()
	return cluster.modCount
}

// The URL is the nominal URL of the cluster itself.  It does
// not have to be syntactically related to its pod URLs
//////func NewInMemoryCluster(url string) (cluster *Cluster) {
func NewInMemoryCluster() (cluster *Cluster) {
	cluster = &Cluster{}
	// cluster.url = url
	cluster.pods = make(map[string]*Pod)
	cluster.modified = sync.NewCond(&cluster.modlock)
	// and as a page?
	// leave that stuff zero for now
	return
}

func (cluster *Cluster) Pods() (result []*Pod) {
	cluster.rlock()
	defer cluster.runlock()
	result = make([]*Pod, 0, len(cluster.pods))
	for _, k := range cluster.pods {
		result = append(result, k)
	}
	return
}

var NameAlreadyTaken = errors.New("URL already taken")

func (cluster *Cluster) AddPod(pod interface{}) error {

	mpod := pod.(*Pod)

	cluster.lock()
	defer cluster.unlock()
	return cluster.locked_AddPod(mpod)
}

func (cluster *Cluster) locked_AddPod(pod *Pod) error {

	url := pod.URL()
	if _, existed := cluster.pods[url]; existed {
		return NameAlreadyTaken
	}

	// use a SetClusterPointer() so pod can be an interface?
	// ... except that might might folks think it's safe to
	// call it themselves
	pod.cluster = cluster

	cluster.pods[url] = pod
	pod.save()

	for _, page := range pod.loadedPages() {
		cluster.clusterTouched(page)
	}

	return nil
}

func (cluster *Cluster) PodByURL(url string) (pod *Pod) {
	cluster.rlock()
	pod = cluster.pods[url]
	cluster.runlock()
	return
}

func (cluster *Cluster) PageByURL(url string, mayCreate bool) (page *Page, created bool) {
	// if we had a lot of pods we could hardcode some logic about
	// what their URLs look like, but for now this should be fine.
	trace("Cluster.PageByURL(%v)", url)
	cluster.rlock()
	defer cluster.runlock()
	for _, pod := range cluster.pods {
		trace("Cluster.PageByURL maybe pod %v", pod.urlWithSlash)
		if strings.HasPrefix(url, pod.urlWithSlash) {
			page, created = pod.PageByURL(url, mayCreate)
			return
		}
	}
	log.Printf("can't do PageByURL -- no suitable pod: %q:", url)
	trace("Cluster.PageByURL(%v) -- NO POD MATCHES", url)
	return
}

// hide these from the public
func (cluster *Cluster) rlock() {
	cluster.mutex.RLock()
}
func (cluster *Cluster) runlock() {
	cluster.mutex.RUnlock()
}
func (cluster *Cluster) lock() {
	cluster.mutex.Lock()
}
func (cluster *Cluster) unlock() {
	cluster.mutex.Unlock()
}

// so that we can access this via an interface
func (cluster *Cluster) AddListener(l chan interface{}) {
	cluster.Listeners.Add(l)
}
