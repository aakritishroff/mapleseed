package inmem


type PageFilter func (page *Page) bool

func (cluster *Cluster) CollectMatchingPages(filter PageFilter, results *[]*Page) {
    cluster.rlock()
    defer cluster.runlock()
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
    page.mutex.RLock()
    defer page.mutex.RUnlock()
    if filter(page) {
        *results = append(*results, page)
    }
}




// as per http://golang.org/pkg/sort/ example
type ByURL []*Page
func (a ByURL) Len() int           { return len(a) }
func (a ByURL) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByURL) Less(i, j int) bool { return a[i].URL() < a[j].URL() }

// to be more flexible we have to wrap them...
// http://play.golang.org/p/4PmJVi2_7D
