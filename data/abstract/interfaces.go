package abstract

import (
	"../inmem"
)

type JSON map[string]interface{}

type Callback *func(interface{})

type Notifier interface {
	AddCallback(cb *func(interface{}))
}

type Page interface {
	Notifier
	Get(prop string) (value interface{}, exists bool)
	GetDefault(prop string, def interface{}) (value interface{})
	NakedGet(prop string) (value interface{}, exists bool)
	NakedGetDefault(prop string, def interface{}) (value interface{})
	SetProperties(m map[string]interface{}, onlyIfMatch string) (etag string, notMatched bool)
	Set(prop string, value interface{})
	AddListener(chan interface{})
	SetContent(contentType string, content string, onlyIfMatch string) (etag string, notMatched bool)
	Content(accept []string) (contentType string, content string, etag string)

	// not yet tested from here on...

	Path() string
	URL() string
	LastModifiedAtClusterModCount() uint64
	// Pod() Site
	Delete()
	Deleted() bool
	WaitForNoneMatch(etag string)
	Properties() (result []string)
	MarshalJSON() (bytes []byte, err error)
	AsJSON() map[string]interface{}
}

type Pod interface {
	Notifier
	AddListener(chan interface{})
	// I still don't quite grok interfaces.  I want this to be returning
	// a Page, but I can't...
	NewPage() (*inmem.Page, string)  
}
 
func NewPod(url string) *inmem.Pod {
	return inmem.NewPod(url)
}

type WebView interface {
	AddPod(pod interface{}) error
}

func NewWebView() WebView {
	return inmem.NewInMemoryCluster()
}

type Listener chan interface{}

func NewPage(impl string) (page Page, etag string) {
	if impl == "inmem" {
		return inmem.NewPage()
	} else {
		panic("unknown implementation requested")
	}
}

