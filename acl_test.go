package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	//"fmt"
	//db "github.com/aakritishroff/datapages/inmem"
	"log"
	"testing"
)

var origin = "http://localhost/"
var url = "ws://localhost:8080/.well-known/podsocket/v1"

func setupUser(username string, data []string) *websocket.Conn {
	dataMap := make(map[string]interface{})
	dataMap["userId"] = username //"aakriti/"
	opLoginData, _ := json.Marshal(InMessage{Seq: 1, Op: "login", Data: dataMap})

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	_, err = ws.Write(opLoginData)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Logging in as %s\n", username)

	//Create a few Pages
	dataMap["initialData"] = data[0] //"Public R+W Hello World!"
	opCreatePageData, _ := json.Marshal(InMessage{Seq: 1, Op: "create", Data: dataMap})
	_, err = ws.Write(opCreatePageData)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Creating pages.. First Page Sent: %s\n", opCreatePageData)

	dataMap["initialData"] = data[1] //"Public R+W Hello World Once Again!"
	opCreatePageData, _ = json.Marshal(InMessage{Seq: 1, Op: "create", Data: dataMap})
	_, err = ws.Write(opCreatePageData)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Creating pages.. Second Page Sent: %s\n", opCreatePageData)

	return ws

}

func Test_isReadableACL(t *testing.T) {

	wsAlice := setupUser("alice/", []string{"Alice's Public Page 0 ", "Alice's Public Page 1"})

	dataMap := make(map[string]interface{})
	dataMap["_id"] = "alice/a0"
	opReadPage, _ := json.Marshal(InMessage{Seq: 1, Op: "read", Data: dataMap})

	_, err := wsAlice.Write(opReadPage)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Sending Read Req... %s\n", opReadPage)
}

func Test_isWritableACL(t *testing.T) {
	wsAlice := setupUser("alice/", []string{"Alice's Public Page 2 ", "Alice's Public Page 3"})
	wsBob := setupUser("bob/", []string{"Bob's Public Page 2", "Bob's Public Page 3"})

	dataMap := make(map[string]interface{})
	dataMap["_id"] = "alice/a0"
	dataMap["_etag"] = 0
	dataMap["randomProp"] = "Booyah"

	opUpdatePage, _ := json.Marshal(InMessage{Seq: 1, Op: "update", Data: dataMap})

	_, err1 := wsBob.Write(opUpdatePage)

	if err1 != nil {
		log.Fatal(err1)
	}

	dataMap = make(map[string]interface{})
	dataMap["_id"] = "alice/a0"
	dataMap["_etag"] = 0
	dataMap["randomProp"] = "Yeah"

	opUpdatePage, _ = json.Marshal(InMessage{Seq: 1, Op: "update", Data: dataMap})

	_, err2 := wsAlice.Write(opUpdatePage)

	if err2 != nil {
		log.Fatal(err2)
	}
	log.Printf("Sending Update Req... %s\n", opUpdatePage)
}

func Test_makePublic(t *testing.T) {

}

/*
setupUser("bob/", []string{"Bob's Public Page", "Bob's Private Page"})

isReadable("alice/", alicePage)
isWritable("alice/", alicePage)

if isReadable("bob/", alicePage) {
	t.Fail()
}

bobPage, _ := cluster.PageByURL("bob/a0", false)
isReadable("bob/", bobPage)
isWritable("bob/", bobPage)

if isReadable("alice/", bobPage) {
	t.Fail()
}*/

/*	c := db.NewInMemoryCluster("http://cluster.as1.crosscloud.org")
	p1, p1x := c.NewPod("http://pod1.as1.crosscloud.org")
	if p1x {
		t.Fail()
	}
	g1, _ := p1.NewPage()
	if g1.URL() != "http://pod1.as1.crosscloud.org/auto/0" {
		t.Fail()
	}*/

/*dataMap = make(map[string]interface{})
dataMap["initialData"] = "Public R+W Hello World Third time's the charm!"
opCreatePage = InMessage{Seq: 1, Op: "create", Data: dataMap}
opCreatePageData, _ = json.Marshal(opCreatePage)
_, err = ws.Write(opCreatePageData)
if err != nil {
	log.Fatal(err)
}

log.Printf("Creating pages.. Third Page Sent: %s\n", opCreatePageData)

var opCreatePageResp = make([]byte, 512)
_, err = ws.Read(opCreatePageResp)
if err != nil {
	log.Fatal(err)
}
log.Printf("Creating pages.. Third Page Received: %s\n", opCreatePageResp)
*/

/*setupUser("alice/", []string{"Alice's Public Page", "Alice's Private Page", "Alice's Shared Page (Alice + Bob)"})
setupUser("bob/", []string{"Bob's Public Page", "Bob's Private Page", "Bob's Shared Page (Alice + Bob)"})
setupUser("eve/", []string{"Eve's Public Page", "Eve's Private Page", "Eve's Shared Readable Page (Alice + Eve)"})
*/
