package abstract

import (
	"testing"
	"time"
	"sync"
	"strconv"
)

func TestMissingValue(t *testing.T) {

	p,_ := NewPage("inmem")
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

	p,_ := NewPage("inmem")
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

	p,_ := NewPage("inmem")
	v1 := "My Value 1"
	v2 := "My Value 2"
	e1a, notMatched := p.SetProperties(JSON{"a":v1}, "wontmatch")
	if notMatched == false {
		t.Error("expected notMatched==true");
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
	l := make(chan interface{},4)
	p1,_ := NewPage("inmem")
	p2,_ := NewPage("inmem")
	p1.AddListener(l)
	p2.AddListener(l)
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

func BenchmarkNakedGetAbs(b *testing.B) {

	p,_ := NewPage("inmem")

	for i := 0; i < b.N; i++ {
		_,_ = p.NakedGet("a")
	}
}


func BenchmarkNakedGetPres(b *testing.B) {

	p,_ := NewPage("inmem")
	v := 1;
	p.Set("a", v)

	for i := 0; i < b.N; i++ {
		_,_ = p.NakedGet("a")
	}
}


func BenchmarkGetAbsent(b *testing.B) {

	p,_ := NewPage("inmem")

	for i := 0; i < b.N; i++ {
		_,_ = p.Get("a")
	}
}

func BenchmarkGetPresent(b *testing.B) {

	p,_ := NewPage("inmem")
	v := 1;
	p.Set("a", v)

	for i := 0; i < b.N; i++ {
		_,_ = p.Get("a")
	}
}


func BenchmarkSetNoChange(b *testing.B) {

	p,_ := NewPage("inmem")
	v := 1;
	p.Set("a", v)

	for i := 0; i < b.N; i++ {
		p.Set("a",v)
	}
}

func BenchmarkSetYesChange(b *testing.B) {
	// why does this take 7x as long?
	// Maybe it's the listeners?

	p,_ := NewPage("inmem")
	v := 1;
	p.Set("a", v)

	for i := 0; i < b.N; i++ {
		p.Set("a",i)
	}
}

func BenchmarkSetYesChange10(b *testing.B) {

	p,_ := NewPage("inmem")

	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			p.Set("a",j)
		}
	}
}


func BenchmarkListener(b *testing.B) {

	l := make(Listener,4)
	p,_ := NewPage("inmem")
	p.AddListener(l)

	for i := 0; i < b.N; i++ {
		p.Set("a",i)
		if p != <- l {
			b.Error()
		}
	}
}

func BenchmarkCallback(b *testing.B) {

	p,_ := NewPage("inmem")

	var callbackRan bool

	/*
	cb := func(data interface{}) {
		// page := data.(Page)
		callbackRan = true
	}
	p.AddCallback(&cb)
*/

	for i := 0; i < b.N; i++ {
		callbackRan = false
		p.Set("a",i)
		if callbackRan != false  {
			b.Error()
		}
	}

}

func BenchmarkCallbackNest(b *testing.B) {

	p,_ := NewPage("inmem")
	count := b.N
	cb := func(data interface{}) {
		// page := data.(Page)
		if count > 0 {
			count--
			p.Set("a",count)
		}
	}
	p.AddCallback(&cb)
	p.Set("a",count)
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

func BenchmarkPageLife(b *testing.B) {

	pod := NewPod("http://example.com/")

	for i := 0; i < b.N; i++ {
		p,_ := pod.NewPage()
		p.Delete()
	}
}




/*
    Run a bunch of goroutines each trying to increment a number stored
    as a string in a page, using etags to handle concurrency.
    Actually works....

*/
func incrementer(page Page,times int,sleep time.Duration,wg *sync.WaitGroup,id int) {
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
	page,_ := NewPage("inmem")
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


