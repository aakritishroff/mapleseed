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



/*

BenchmarkSetNoChange	10000000	       205 ns/op
BenchmarkSetYesChange	 1000000	      1547 ns/op
BenchmarkSetYesChange2	  500000	      3045 ns/op
BenchmarkSetYesChange3	  500000	      3029 ns/op
BenchmarkGetPresent	20000000	           140 ns/op
BenchmarkGetPresentNoLock	50000000	    42 ns/op
BenchmarkGetAbsent	20000000	           125 ns/op
BenchmarkListener	 1000000	          1742 ns/op
BenchmarkPodNewPage	  500000	          4348 ns/op

I wonder how it would compare if instead each pod were
a goroutine, so there was no locking...

BenchmarkChannel	 5000000	       506 ns/op
BenchmarkChannel2	 5000000	       445 ns/op

Ah, no, that's a lot more than the overhead of locking.  Okay,
I guess we'll live with 100s of ns per operation.  Maybe we'll
want different granularity....

*/


func BenchmarkSetNoChange(b *testing.B) {

	p,_ := NewPage()
	v := 1;
	p.Set("a", v)

	for i := 0; i < b.N; i++ {
		p.Set("a",v)
	}
}

func BenchmarkSetYesChange(b *testing.B) {
	// why does this take 7x as long?
	// Maybe it's the listeners?

	p,_ := NewPage()
	v := 1;
	p.Set("a", v)

	for i := 0; i < b.N; i++ {
		p.Set("a",i)
	}
}

func BenchmarkSetYesChange2(b *testing.B) {

	p,_ := NewPage()

	for i := 0; i < b.N; i++ {
		p.Set("a",1)
		p.Set("a",2)
	}
}
func BenchmarkSetYesChange3(b *testing.B) {

	p,_ := NewPage()

	for i := 0; i < b.N; i++ {
		p.Set("a",1)
		p.Set("a",2)
	}
}

func BenchmarkGetPresent(b *testing.B) {

	p,_ := NewPage()
	v := 1;
	p.Set("a", v)

	for i := 0; i < b.N; i++ {
		_,_ = p.Get("a")
	}
}
func BenchmarkGetPresentNoLock(b *testing.B) {

	p,_ := NewPage()
	v := 1;
	p.Set("a", v)

	for i := 0; i < b.N; i++ {
		_,_ = p.locked_Get("a")
	}
}

func BenchmarkGetAbsent(b *testing.B) {

	p,_ := NewPage()

	for i := 0; i < b.N; i++ {
		_,_ = p.Get("a")
	}
}


func BenchmarkListener(b *testing.B) {

	l := make(Listener,4)
	p,_ := NewPage()
	p.Listeners.Add(l)

	for i := 0; i < b.N; i++ {
		p.Set("a",i)
		if p != <- l {
			b.Error()
		}
	}
}

func echo(to, from chan string) {
	for {
		message := <- to
		if message == "" {
			return
		}
		from <- message
	}
}

func BenchmarkChannel(b *testing.B) {

	to := make(chan string)
	from := make(chan string)

	go echo(to, from)

	for i := 0; i < b.N; i++ {
		to <- "hello world!"
		_ = <- from
	}
	to <- ""
}

func BenchmarkChannel2(b *testing.B) {

	to := make(chan string, 2)
	from := make(chan string, 2)

	go echo(to, from)

	for i := 0; i < b.N; i+=2 {
		to <- "hello world!"
		to <- "hello world!"
		_ = <- from		
		_ = <- from
	}
	to <- ""
}
