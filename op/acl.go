/*
ACL module that manages permissions on datapages.
Currently represented as an ACL struct and stored in
the datapage's
page.appData["isPublic"] --> true/false and
page.appData["readers"] -->[]string (== userIDs of authorized readers)

*/

package op

import (
	db "github.com/aakritishroff/mapleseed/data/inmem"
	"log"
)

var propPub = "isPublic"    //constant variable used to access acl property in datapage's appData
var propReaders = "readers" //constant variable used to access acl property in datapage's appData
var propOwner = "_owner"

/*
If owner doesn't set isPublic property during creation, defaults to true.
If owner sets isPublic to false, only owner and userIDs in set of "readers" are allowed to read.
*/
/*func createACL(page *db.Page, owner string, isPub bool) {
	page.Set(propPub, isPub)

	oldReaders, ok := page.Get(propReaders)
	if !ok {
		page.Set(propReaders, []string{owner})
	} else {
		newReaders := oldReaders.([]string)
		newReaders = append(newReaders, owner)
		page.Set(propReaders, newReaders)
		log.Printf("Created ACL... Owner: %s, isPublic: %t, Readers: %s", owner, isPub, newReaders)
	}

}*/

/*
Checks if user with userID can read datapage
*/
func isReadable(userID string, page *db.Page) bool {
	isPublic, readers, ok := getACL(page)
	owner, _ := page.Get(propOwner)

	if userID == owner.(string) {
		return true
	}

	if !ok {
		log.Printf("getACL did not find ACL. Check code..")
		return false
	}

	if isPublic {
		return true
	}
	for _, v := range readers {
		if v == userID {
			return true
		}

	}
	return false //not found in readers and isPublic == false

}

/*
Checks if user with userID can write datapage. Must be owner
*/
func isWritable(userID string, page *db.Page) bool {
	val, ok := page.Get("_owner")
	if ok {
		owner := val.(string)
		return owner == userID
	}
	return false
}

/*
Get ACL for datapage
ok = false iff isPublic false, but "readers" non-existant
*/
func getACL(page *db.Page) (isPublic bool, readers []string, ok bool) {
	valPub, existsPub := page.Get(propPub)
	valReaders, existsReaders := page.Get(propReaders)
	ok = false
	if existsPub {
		isPublic = valPub.(bool)
		if isPublic {
			readers = []string{}
			ok = true
		}
	}
	if existsReaders {
		readers = valReaders.([]string)
		ok = true
	}
	return
	/*if exists1 {
		isPublic = val1.(bool)
		if !isPublic {
			val2, exists2 := page.Get(propReaders)
			if exists2 {
				readers = val2.([]string)
				ok = true
			} else {
				log.Println("isPublic is false, but readers property not found! Shouldn't reach here!")
				ok = false
			}
		} else {
			readers = []string{} //isPublic = true, readers = empty slice
			ok = true
		}
	} else {
		log.Println("isPublic property not found! Either page doesn't exist or shouldn't reach here!")
		ok = false
	}
	return*/

}

/*
Reset ACL, remove all readers except owner
*/
func resetACL(page *db.Page, owner string) {
	page.Set(propPub, false)
	page.Set(propReaders, []string{owner})
}

/*
Adds userID to readers and make sure isPublic == false
*/
func addReader(userID string, page *db.Page) {
	page.Set(propPub, false)
	oldReaders, ok := page.Get(propReaders)
	if !ok {
		page.Set(propReaders, []string{userID})
	} else {
		newReaders := oldReaders.([]string)
		newReaders = append(newReaders, userID)
		page.Set(propReaders, newReaders)
	}
}

/*
Revokes userID
*/
func revokeReader(userID string, page *db.Page) {
	_, readers, ok := getACL(page)
	if ok {
		newReaders := []string{}
		for _, v := range readers {
			if v != userID {
				newReaders = append(newReaders, v)
			}
		}
	}
}
