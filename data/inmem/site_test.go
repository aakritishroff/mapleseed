package inmem

import (
	"testing"
)

func BenchmarkPodNewPage(b *testing.B) {

	cluster := NewInMemoryCluster()
	pod  := NewPod("http://example.com/")
	cluster.AddPod(pod)

	for i := 0; i < b.N; i++ {
		pod.NewPage()
	}
}
