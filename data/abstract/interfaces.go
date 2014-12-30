package abstract

import (
	"../inmem"
)

type JSON map[string]interface{}

type Page interface {
	Get(prop string) (value interface{}, exists bool)
	GetDefault(prop string, def interface{}) (value interface{})
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
	Deleted() bool
	WaitForNoneMatch(etag string)
	Properties() (result []string)
	MarshalJSON() (bytes []byte, err error)
	AsJSON() map[string]interface{}
}

type Site interface {
	// ...?
}

type Listener chan interface{}

func NewPage(impl string) (page Page, etag string) {
	if impl == "inmem" {
		return inmem.NewPage()
	} else {
		panic("unknown implementation requested")
	}
}

