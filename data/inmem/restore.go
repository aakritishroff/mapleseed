package inmem

import (
	"errors"
    "log"
	"time"
	"io"
	"regexp"
	"encoding/json"
)

type JSON map[string]interface{}

func (cluster *Cluster) RestoreFrom(src io.Reader) error {
	cluster.lock()   // no one better touch anything while we're doing this!
	defer cluster.unlock()

	dec := json.NewDecoder(src)
	var v JSON
	if err := dec.Decode(&v); err != nil {
		log.Println(err)
		return err
	}

	etag, has_etag := v["_etag"]
	if has_etag {   // version 1 dump format, used by fakepods.go
		cluster.modCount = uint64(etag.(float64))
		log.Printf("etag %d", cluster.modCount)
		
		members := v["_members"].([]interface{})
		
		// doesn't work for localhost, etc
		re := regexp.MustCompile("^(http://([^.]+).[^.]+\\.com)(.*)$")
		
		for i,obj := range members {
			log.Printf("restoring member %d: %q\n\n", i, obj)
			pmap := obj.(map[string]interface {})
			id := pmap["_id"].(string)
			
			m := re.FindStringSubmatch(id)
			podId := m[1]
			if podId != pmap["_owner"] {
				return errors.New("page has wrong owner: "+id)
			}
			// userName := m[2]
			path := m[3]
			pod := NewPod(podId)
			err := cluster.AddPod(pod)
			if err != nil {
				return errors.New("failed to build add: "+podId)
			}
			page,created := pod.PageByPath(path, true)
			if !created {
				return errors.New("page already existed: "+id)
			}
			
			page.locked_OverlayWithMap(pmap)
			
			page.modCount = uint64(pmap["_etag"].(float64))
			
			lastModString, present := pmap["_lastModified"]
			if present {
				var err error
				page.lastModified, err = time.Parse(time.RFC3339Nano, lastModString.(string))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
