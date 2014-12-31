package inmem

// How is this going to work with vocabspec?   Somehow the mastervs
// has to be maintained as well....


import (
	"errors"
	"net/url"
	"os"
	"strings"
	"encoding/json"
	"log"
	"strconv"
	"io/ioutil"
)

// FSBind mirrors the data to/from a directory in the filesystem
//
// The dirname is the name of that directory.  The top level is the
// names of all the pods, then the pages under them.  FSBind first
// loads any/all pods, and makes it so page requests will load pages
// on demand.  It also makes it so all future pod creations and page
// modifications will be writen to disk.   
//
// It assumes it's the only thing writing to disk.  It wont read from
// the disk once it has something in memory until a restart.
func (cluster *Cluster) FSBind(dirname string) {
	cluster.fsroot = dirname    // this is the flag to be writing stuff
	os.MkdirAll(dirname, 0700)
	cluster.recreatePodsFromDisk()

	// I've no real clue how big to make this queue, but we'd like
	// to minimize blocking on disk-writes, so we want this fairly
	// big.   Note that as the writer gets behind, it'll be skipping
	// intermediate writes, so it has some chance of catching up.
	//
	// synthetic benchmark, just setting 1 var over and over (ie the
	// case where a large queue helps the most), time for page.Set():
	// go test -test.bench=YesCh (this is WITHFS)
	//
	//    q size     time
	//      1         172014 ns/op
    //      2         84920 ns/op
    //      100       2608 ns/op
    //      1k        1130 ns/op
	//      5k        974 ns/op
    //      10k       956 ns/op
	dirtyQueue := make(Listener, 5000)
	cluster.queueForFSB = dirtyQueue // ehhh, I didnt want to leak this

	cluster.AddListener(dirtyQueue)
	go func() {
		for {
			msg := <- dirtyQueue
			switch msg := msg.(type) {
			case *Page:
				// skip the save if we've already saved it, because it
				// ended up on the queue multiple times
				if msg.modCount > msg.lastSaved {
					msg.save()
				}
			case chan bool:
				// if someone sends us a (chan bool), then send them
				// back a true.  This allows someone to find out when
				// we get through the queue.
				msg <- true
			}
		}
	}()
}

// Flush returns when every page that's currently in the dirtyQueue
// has been saved to disk.   More pages might have been put in the 
// queue during that time, but we know how far we got.  Maybe we should
// also call os.(*File).Sync(), but ... on which file?
func (cluster *Cluster) Flush() {
	relay := make(chan bool)
	cluster.queueForFSB <- relay
	_ = <- relay
}

func (page *Page) filename() string {
	return page.pod.filename()+"/pg_"+url.QueryEscape(page.path)
}

func (pod *Pod) pathTakenOnDisk(path string) bool {
	filename := pod.filename()+"/"+url.QueryEscape(path)
	fi,err := os.Open(filename)
	if err == nil {
		fi.Close()
		return true
	} else if os.IsExist(err) {
		return false
	}
	panic(err)
}

func (pod *Pod) filename() string {
	return pod.cluster.fsroot+"/"+url.QueryEscape(pod.TrimmedName())
}

// trimmedName give back the pod URL after removing the leading
// "http[s]://" and trailing "/"
func (pod *Pod) TrimmedName() string {
	name := pod.urlWithSlash
	if !strings.HasSuffix(name, "/") {
		panic("missing trailing slash: "+name)
	}
	name = name[:len(name)-1]
	if strings.HasPrefix(name, "http://") {
		name = name[7:]
	} else if strings.HasPrefix(name, "https://") {
		name = name[8:]
	} else {
		panic("weird pod name: "+pod.urlWithSlash)
	}
	return name
}





// load fetches the contents of the page from disk, if an fsroot has
// been set, there's content there, etc.  pod.PageByPath calls this
// if a page isn't found, so that in memory pages are essentially just
// a cache (although they are the master when they exist).
//
// it's not an error for load to not find anything, so we have to return
// parms
//
func (page *Page) load() (loaded bool, err error) {
	if page.pod == nil || page.pod.cluster == nil || page.pod.cluster.fsroot == "" {
		return false, nil
	}

	filename := page.filename()
	src, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer src.Close()
	dec := json.NewDecoder(src)
	var obj JSON
	if err := dec.Decode(&obj); err != nil {
		log.Println("BAD JSON restoring from ", filename)
		log.Println(err)  // how did bad JSON get here?
		return false, err
	}
	// pmap := obj.(map[string]interface {})
	pmap := obj
	// id := pmap["_id"].(string)

	//log.Println("Got JSON", pmap)
	etag := pmap["_etag"].(string)
	modCount,_ := strconv.ParseUint(etag, 10, 64)
	delete(pmap, "_etag")
	page.SetProperties(pmap, "")
	page.modCount = modCount
	page.lastSaved = modCount
			
	return true, nil
}

// save should only be called by MirrorToDisk, so it'll never be
// concurrent.   Well, it would be okay to run multiple mirrorToDisk-like
// operations on different spindles, I suppose....
func (page *Page) save() error {
	if page.pod == nil || page.pod.cluster == nil || page.pod.cluster.fsroot == "" {
		return errors.New("not configured to load page")
	}

	filename := page.filename()
	tfilename := filename + "_tmp"

	dst, err := os.Create(tfilename)
	if err != nil {
		panic(err)
	}
	data, modCount := page.AsJSONWithModCount()
	delete(data, "_id")
	delete(data, "_owner")
	enc := json.NewEncoder(dst)
	enc.Encode(data)
	if err != nil {
		panic(err)
	}
	err = dst.Close()
	if err != nil {
		panic(err)
	}
	err = os.Remove(filename) 
	if err != nil {
		if os.IsNotExist(err) {
			// this is fine
		} else {
			panic(err)
		}
	}
	err = os.Rename(tfilename, filename)
	if err != nil {
		panic(err)
	}
	page.lastSaved = modCount
	return nil
}



func (pod *Pod) save() {
	if pod.cluster != nil && pod.cluster.fsroot != "" {
		err := os.MkdirAll(pod.filename(), 0700)
		if err != nil {
			if os.IsExist(err) {
				// pass
			} else {
				panic(err)
			}
		}
		pwfile := pod.filename() + "/pw"
		err = ioutil.WriteFile(pwfile, pod.pwHash, 0600)
		if err != nil {
			panic(err)
		}
	}
}

func (pod *Pod) loadAllPages() {
	//log.Printf("loadAllPages for %q",pod.TrimmedName())
	dir, err := os.Open(pod.filename())
	if err != nil {
		panic(err)
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		panic(err)
	}
	pages := make(map[string]*Page)
	for _,name := range names {
		//filename := pod.filename()+"/"+name
		page,_ := NewPage()
		page.path,err = url.QueryUnescape(name)
		//log.Printf(".. loading %q", page.path)
		page.pod = pod
		loaded, err := page.load()
		if err != nil {
			panic(err)
		}
		if !loaded {
			panic("page I was trying to load disappeared:"+ name)
		}
		pages[page.path] = page
	}

	// we only need it locked when we're copying these all in...
    pod.Lock()
	defer pod.Unlock()
	for path,page := range pages {
		oldPage,_ := pod.pages[path]
		if oldPage == nil {
			pod.pages[path] = page
		} else {
			log.Printf("New page was created while loading same URL from disk");
		}
	}
	pod.fullyLoaded = true
}

func (cluster *Cluster) recreatePodsFromDisk() {
	cluster.lock()
	defer cluster.unlock()
	
	root, err := os.Open(cluster.fsroot)
	if err != nil {
		panic(err)
	}
	defer root.Close()
	names,err := root.Readdirnames(-1)
	if err != nil {
		panic(err)
	}
	for _,name := range names {

		realname,err := url.QueryUnescape(name)
		url := "http://"+realname+"/"
		if err != nil {
			panic(err)
		}
		pod := NewPod(url)
		pod.pwHash, err = ioutil.ReadFile(pod.filename()+"/pw")
		if err != nil {
			panic(err)
		}
		cluster.AddPod(pod)
		log.Printf("restored pod %q from disk", name)
	}
}
