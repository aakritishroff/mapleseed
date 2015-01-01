package inmem

import (
	"testing"
	"os"
	//"log"
)

func TestFS1(t *testing.T) {
	testDir := "./.fsbind-test"
	podurl := "http://foo.example/bar/"
	w := NewInMemoryCluster()
	os.RemoveAll(testDir)
	// defer os.RemoveAll(testDir)
	w.FSBind(testDir)
	p := NewPod(podurl)
	err := w.AddPod(p)
	if err != nil {
		t.Error()
		return
	}
	pg,_ := p.NewPage()
	pg.Set("a", "100");
	id := pg.URL()
	w.Flush() // make sure everything's written before we try to re-read it
	w = nil
	p = nil
	pg = nil

	w = NewInMemoryCluster()
	w.FSBind(testDir)
	//log.Printf("w.pods after FSBind %q", w.pods)
	p = w.PodByURL("http://foo.example/bar/")
	if p == nil {
		t.Error("expected p to exist, but it didn't")		
		return
	}
	//log.Printf("%q", p.fullyLoaded)
	pg,created := w.PageByURL(id, false)
	if pg == nil {
		t.Error()
		return
	}
	if created {
		t.Error()
		return
	}
	//log.Printf("pg: %q", pg)
	if pg.GetDefault("a", nil) != "100" {
		t.Error()
		return
	}
}


func BenchmarkSetYesChangeWITHFS(b *testing.B) {
	testDir := "./.fsbind-test"
	podurl := "http://foo.example/bar/"
	w := NewInMemoryCluster()
	os.RemoveAll(testDir)
	w.FSBind(testDir)
	pod := NewPod(podurl)
	err := w.AddPod(pod)
	if err != nil {
		b.Error()
	}

	p,_ := pod.NewPage()
	v := 1;
	p.Set("a", v)

	b.ResetTimer() // rmdir might have taken a long time

	for i := 0; i < b.N; i++ {
		p.Set("a",i)
	}
	w.Flush()
}
func BenchmarkSetYesChangeWITHFSAndFlush(b *testing.B) {
	testDir := "./.fsbind-test"
	podurl := "http://foo.example/bar/"
	w := NewInMemoryCluster()
	os.RemoveAll(testDir)
	w.FSBind(testDir)
	pod := NewPod(podurl)
	err := w.AddPod(pod)
	if err != nil {
		b.Error()
	}

	p,_ := pod.NewPage()
	v := 1;
	p.Set("a", v)

	b.ResetTimer() // rmdir might have taken a long time

	for i := 0; i < b.N; i++ {
		p.Set("a",i)
		w.Flush()
	}
}
