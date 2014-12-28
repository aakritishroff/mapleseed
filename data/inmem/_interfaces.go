/*

   If golang had return type covariance (like c++ templates) then this
   could actually be the declared interface to all pod types.  Instead,
   this is just documentation, really.

   Key design point: we're arms length from the data so this can be
   threadsafe and also can be run across an unreliable network link
   (where locking is out of the question).

   ? add user-identity?

   The key concept is that a Pod is a set of Pages owned by a single
   users.  Pods are grouped together into a Cluster, which might be a
   server or set of servers.

*/

package db

type Page interface {

	// Get a string that uniquely identifies this page, eg its URL
	// Note: page.Pod().Id() must be a leading substring of this.
	Id() string

	// Access the Pod which contains this page
	Pod() Pod

	// Atomically obtain a copy of the page content
	// Might be sub-selected in some way (eg Accept or Range or Filter)
	Content(selector ...interface{}) (content interface{}, etag string, []selectionDetails)

	// Atomically set the page content
	SetContent(content interface{}, onlyIfMatch string) (etag string, notMatched bool)

	WaitForChange(etag string)

	AccessControlPage() AccessController // or is this just a Page

	// forget state, and cluster.GetPage(id) wont return it without
	// re-creating it
	Delete()
	
	// some pages have content, some have properties, some have both
	// "RDF Source" pages are dataOnly, content is nil.

	// Get a copy of all the names of properties of the page
	Properties() []string

	// Atomically obtain a copy of the value of some named page property
	Get(prop string) (value interface{}, exists bool)

	// Atomically set the value of some named page property
	Set(prop string, value interface{})

	// needs vocab handling at some point ... 
	// also, this isn't quite right, given how the nesting works.
	// Maybe we need  Page.triples  ?   Page.db ?
}

type ExternalUserId string

type AccessController interface {
	DataPage
	ReadableBy(indiv ExternalUserId) bool
	SetReadableBy(indiv ExternalUserId)
	ClearReadableBy(indiv ExternalUserId)
	Readers() []ExternalUserId
}

type Pod interface {     // aka Container, for these purposes?
	Page
	Pages(filter ...interface{}) []*Page    
	NewPage() (page *Page)
	GetPage(id string) (page *Page)  // returns nil if fails
	GetPage(id string, mayCreate bool) (page *Page, created bool)
}

type Cluster interface {
	Page
	Pods() []*Pod
	Pages(filter ...interface{}) []*Page    
	GetPod(id string, mayCreate bool) (pod *Pod, existed bool)
	GetPage(id string, mayCreate bool) (page *Page, created bool)
}

type Modification {
	/*
	page creation
    content replaced
    set property   (compare add/delete triples)
	page deletion

	... is a Pod anything special in this regard?  
	   
	really, the only thing a pod does for us is be a container...

    (a nestable container)


   */
}
