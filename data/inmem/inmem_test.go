package inmem

import (
	"testing"
	"fmt"
	"time"
	"strconv"
	"sync"
	"encoding/json"
)

func TestVeryBasic(t *testing.T) {

	// ridiculous test, hard codes our page URL generation algorithm

	c := NewInMemoryCluster("http://cluster.example/")
	p1, p1x := c.NewPod("http://pod1.example/")
	if p1x { t.Fail() }
	if p1 == nil { t.Fail() }
	g1,_ := p1.NewPage()
	if g1.URL() != "http://pod1.example/a0" { t.Fail() }
	g2,_ := p1.NewPage()
	if g2.URL() != "http://pod1.example/a1" { t.Fail() }
	g2.Delete()
	g3,_ := p1.NewPage()
	if g3.URL() != "http://pod1.example/a2" { t.Fail() }
}

/*
    Run a bunch of goroutines each trying to increment a number stored
    as a string in a page, using etags to handle concurrency.
    Actually works....

*/
func incrementer(page *Page,times int,sleep time.Duration,wg *sync.WaitGroup,id int) {
	//fmt.Printf("incr started\n");
	for i:=0; i<times; {
		_,content,etag := page.Content([]string{})
		n,_ := strconv.ParseUint(content, 10, 64)
		n++
		content = strconv.FormatUint(n, 10)
		time.Sleep(sleep)
		_, notMatched := page.SetContent("", content, etag)
		if notMatched {
			// fmt.Printf("was beaten to %d\n", n);
		} else {
			//fmt.Printf("%d did incr to %d\n", id, n);
			i++
		}
	}
	wg.Done()
	//fmt.Printf("done\n");
}

func TestIncr1(t *testing.T) {
	c := NewInMemoryCluster("http://cluster.example")
	pod,_ := c.NewPod("http://pod1.example")
	page,_ := pod.NewPage()
	_,_ = page.SetContent("", "1000", "")
	var wg sync.WaitGroup
	for i:=0; i<10; i++ {
		wg.Add(1)
		go incrementer(page,10,5*time.Millisecond,&wg,i)
	}
	wg.Wait()
	_,content,_ := page.Content([]string{})
	// fmt.Printf(content)
	if content != "1100" { t.Fail() }
}


func _TestJSON(t *testing.T) {

	c := NewInMemoryCluster("http://cluster.example/")
	b,err := json.Marshal(c)
	if err != nil { 
		fmt.Printf("error: %q\n", err)
		t.Fail() 
		return
	}
	fmt.Printf("cluster = %q\n\n",b) 

	p1, _ := c.NewPod("http://pod1.example/")
	b,err = json.Marshal(p1)
	if err != nil { 
		fmt.Printf("error: %q\n", err)
		t.Fail() 
		return
	}
	fmt.Printf("json: %q\n", b)
	
}


