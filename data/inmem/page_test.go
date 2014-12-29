package inmem

import (
	"testing"
	"time"
	"sync"
	"strconv"
)

func TestMissingValue(t *testing.T) {

	p,_ := NewPage()
	val, exists := p.Get("a")
	if val != nil {
		t.Error("expected nil from missing value, got %q", val)
	}
	if exists {
		t.Error("expected exists to be false")
	}
	if p.GetDefault("a", "default").(string) != "default" {
		t.Error("expected default value")
	}

}

func TestStringValue(t *testing.T) {

	p,_ := NewPage()
	v := "My Value"
	p.Set("a", v)
	val, exists := p.Get("a")
	if val != v {
		t.Error("unexpected value, got %q, not %q", val, v)
	}
	if !exists {
		t.Error("expected exists to be true")
	}
	if p.GetDefault("a", "default") != v {
		t.Error("did not expect default value")
	}

}


func TestETags(t *testing.T) {

	p,_ := NewPage()
	v1 := "My Value 1"
	v2 := "My Value 2"
	e1a, notMatched := p.SetProperties(JSON{"a":v1}, "wontmatch")
	if notMatched == false {
		t.Error("expected notMatched==true");
	}
	e1b := p.etag()
	if e1a != e1b {
		t.Error("etags should be the same");
	}
	e2, notMatched := p.SetProperties(JSON{"a":v1}, "")
	if notMatched {
		t.Error("expected notMatched==false");
	}
	if (e1a == e2) {
		t.Error("etag should be different now, but %q == %q", e1a, e2)
	}
	e3, notMatched := p.SetProperties(JSON{"a":v2}, e2)
	if notMatched {
		t.Error("expected notMatched==false");
	}
	if (e3 == e2) {
		t.Error("etag should be different now, but %q == %q", e2, e3)
	}

}

func TestListener(t *testing.T) {
	l := make(Listener,4)
	p1,_ := NewPage()
	p2,_ := NewPage()
	p1.Listeners.Add(l)
	p2.Listeners.Add(l)
	v1 := "My Value 1"
	v2 := "My Value 2"
	p1.Set("a", v1)
	p2.Set("a", v1)
	p1.Set("a", v2)
	p2.Set("a", v2)
	if p1 != <- l {
		t.Error()
	}
	if p2 != <- l {
		t.Error()
	}
	if p1 != <- l {
		t.Error()
	}
	if p2 != <- l {
		t.Error()
	}
	l <- nil
	if nil != <- l {
		t.Error()
	}
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
			// fmt.Printf("%d did incr to %d\n", id, n);
			i++
		}
	}
	wg.Done()
	//fmt.Printf("done\n");
}

func TestIncr1(t *testing.T) {
	page,_ := NewPage()
	_,_ = page.SetContent("", "1000", "")
	var wg sync.WaitGroup
	for i:=0; i<10; i++ {
		wg.Add(1)
		go incrementer(page,10,10*time.Microsecond,&wg,i)
	}
	wg.Wait()
	_,content,_ := page.Content([]string{})
	// fmt.Printf(content)
	if content != "1100" { t.Error("content was", content) }
}
