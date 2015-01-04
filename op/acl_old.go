/*
ACL module that manages permissions on datapages.
Currently represented as an ACL struct and stored in the datapage's page.appData["acl"].
*/

package op

// /*/*/*/*/*import (
// 	//"fmt"
// 	db "github.com/aakritishroff/datapages/inmem"
// 	"log"
// )

// /*
// @TODO
// - login as userID "aakriti/" ?œing ACL
// - Need to discuss how UserId == podURL work with cluster. Is podURL the right string for userID?
// - Do we need to lock/unlock ACLS?
// 	It's a part of the datapage and only owner can modify ACL, so probably not
// - ACLs are only modified by the owner... Too strict?

// - userID == pod.URL()
// - If Writable, then readable for now..
// - If isPubReadable/Writable == true, don't check in UserPerms
// - If datapage isPubReadable == true, still need to maintain list of authorized writers in userPerms
// - If datapage isPubWritable == true, implies isPubReadable == true
// */
// œœœ
// type ACL struct {
// 	Page          *db.Page
// 	UserPerms     map[string]*Perm //{"alice": {R: True, W: False}}
// 	Owner         string           //created By
// 	IsPubReadable bool
// 	IsPubWritable bool
// }

// type Perm struct {
// 	R bool
// 	W bool
// }

// var propACL = "acl" //constant variable used to access acl property in datapage's appData

// /*
// Creates ACL for datapage when it datapage is created
// Authorize owner with R+W permissions for datapage
// */
// func createACL(page *db.Page, owner string, isPubW bool, isPubR bool) (acl *ACL) {
// 	acl = &ACL{}
// 	acl.Page = page
// 	acl.Owner = owner
// 	acl.IsPubReadable = isPubR
// 	acl.IsPubWritable = isPubW
// 	acl.UserPerms = map[string]*Perm{}
// 	acl.addPerm(owner, true, true)

// 	setACL(acl, acl.Page)
// 	return
// }

// /*
// Creates ACL for datapage that is public R+W
// */
// func createOpenACL(page *db.Page, owner string) (acl *ACL) {
// 	acl = &ACL{}
// 	acl.Page = page
// 	acl.Owner = owner
// 	acl.IsPubReadable = true
// 	acl.IsPubWritable = true
// 	acl.UserPerms = map[string]*Perm{}
// 	acl.addPerm(owner, true, true)

// 	setACL(acl, acl.Page)
// 	log.Printf("Created ACL... Owner: %s R: %s W: %s", owner, acl.UserPerms[owner].R, acl.UserPerms[owner].W)
// 	return
// }

// /*
// Get ACL for datapage
// */
// func getACL(page *db.Page) (acl *ACL, exists bool) {
// 	var val interface{}
// 	val, exists = page.Get(propACL)
// 	if exists {
// 		acl = val.(*ACL)
// 	}
// 	return
// }

// /*
// Set ACL for datapage
// */
// func setACL(acl *ACL, page *db.Page) {
// 	page.Set(propACL, acl)
// }

// /*
// Reset ACL, remove all permissions
// */
// func resetACL(page *db.Page) {
// 	setACL(&ACL{Page: page}, page)
// }

// /*
// Checks if user with userID can read datapage
// */
// func isReadable(userID string, url string) bool { //page *db.Page
// 	page, _ := cluster.PageByURL(url, false)
// 	acl, exists := getACL(page)
// 	if exists {
// 		if perm, ok := acl.UserPerms[userID]; ok {
// 			return perm.R
// 		}
// 	}
// 	return false
// }

// /*
// Checks if user with userID can write datapage
// */
// func isWritable(userID string, url string) bool {
// 	page, _ := cluster.PageByURL(url, false)
// 	acl, exists := getACL(page)
// 	if exists {
// 		if perm, ok := acl.UserPerms[userID]; ok {
// 			return perm.W
// 		}
// 	}
// 	return false
// }

// /*
// Adds R permission to userID
// */
// func (acl *ACL) addReader(userID string) {
// 	if _, ok := acl.UserPerms[userID]; !ok {
// 		acl.UserPerms[userID] = &Perm{R: false, W: false}
// 	}

// 	acl.UserPerms[userID].R = true
// }

// /*
// Adds W permission to userID
// */
// func (acl *ACL) addWriter(userID string) {
// 	if _, ok := acl.UserPerms[userID]; !ok {
// 		acl.UserPerms[userID] = &Perm{R: false, W: false}
// 	}

// 	acl.UserPerms[userID].R = true
// 	acl.UserPerms[userID].W = true
// }

// /*
// Adds userID to ACL
// */
// func (acl *ACL) addPerm(userID string, permR bool, permW bool) {
// 	acl.UserPerms[userID] = &Perm{R: permR, W: permW}
// }

// /*
// Revokes W permission from userID, may still have R permissions
// */
// func (acl *ACL) revokeWriter(userID string) {
// 	acl.UserPerms[userID].W = false
// }

// /*
// Revokes all permissions from userID
// If R permission is revoked, W is revoked as well
// */
// func (acl *ACL) revokePerm(userID string) {
// 	delete(acl.UserPerms, userID)
// }

// /*
// Make ACL public R+W
// Sets isPubReadable + isPubWritable = true
// */
// func makeOpen(page *db.Page) {
// 	acl, exists := getACL(page)
// 	if exists {
// 		acl.IsPubReadable = true
// 		acl.IsPubWritable = true
// 		for u, _ := range acl.UserPerms {
// 			acl.addPerm(u, true, true)
// 		}
// 	}
// }

// /*
// Make ACL public Readable
// Sets isPubReadable true
// */
// func makeOpenReadable(page *db.Page) {
// 	acl, exists := getACL(page)
// 	if exists {
// 		acl.IsPubReadable = true
// 		for u, _ := range acl.UserPerms {
// 			acl.addReader(u)
// 		}
// 	}
// }

// /*
// Get list of users with R permission
// Returns readers as a map(string)bool to simplify set-like structure in golang
// */
// func (acl *ACL) getReaders() map[string]bool {
// 	readers := map[string]bool{}
// 	for u, p := range acl.UserPerms {
// 		if p.R == true {
// 			readers[u] = true
// 		}
// 	}
// 	return readers
// }

// /*
// Get list of users with W permission
// Returns writers as a map(string)bool to simplify set-like structure in golang
// */
// func (acl *ACL) getWriters() map[string]bool {
// 	writers := map[string]bool{}
// 	for u, p := range acl.UserPerms {
// 		if p.W == true {
// 			writers[u] = true
// 		}
// 	}
// 	return writers
// }
