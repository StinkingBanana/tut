package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Jeffail/gabs"
	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

func backendServer(port string) {
	router := mux.NewRouter()
	router.HandleFunc("/", GetRoot).Methods("GET")
	router.HandleFunc("/followers", GetFollowers).Methods("GET")
	router.HandleFunc("/refollowers", GetReFollowers).Methods("GET")
	router.HandleFunc("/followersID", GetFollowersID).Methods("GET")
	router.HandleFunc("/unfollowers", GetUnfollowers).Methods("GET")
	router.HandleFunc("/user/{id}", GetUser).Methods("GET")
	fmt.Printf("[SYS] Server listening at http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

// GetRoot prints out root message
func GetRoot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(`
	<p>Welcome to Twitch Unfollow Tracker</p>
	<p>Available Endpoints:</p>
	<ul>
	<li><a href="/followers">/followers</a></li>
	<li><a href="/refollowers">/refollowers</a></li>
	<li><a href="/followersID">/followersID</a></li>
	<li><a href="/unfollowers">/unfollowers</a></li>
	<li>/user/{id}</li>
	</ul>
	`))
}

// GetReFollowers find all refollowers detailed info
func GetReFollowers(w http.ResponseWriter, r *http.Request) {
	var outputUsers []User
	db, err := bolt.Open(defaultDBName, 0600, nil)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer db.Close()

	db.View(func(tx *bolt.Tx) error {
		uf := tx.Bucket([]byte("unfollowers"))
		f := tx.Bucket([]byte("followers"))
		u := tx.Bucket([]byte("users"))

		if f != nil && u != nil && uf != nil {
			uf.ForEach(func(k, v []byte) error {
				fdata := f.Get(k)
				if fdata == nil {
					return nil
				}
				udata := u.Get(k)
				if udata != nil && len(udata) > 0 {
					parsed, _ := gabs.ParseJSON(udata)
					jsondata, _ := parsed.ChildrenMap()
					out := User{
						string(k),
						jsondata["login"].Data().(string),
						jsondata["display_name"].Data().(string),
						jsondata["profile_image_url"].Data().(string),
						string(fdata),
						string(v)}
					outputUsers = append(outputUsers, out)
				} else {
					out := User{
						string(k),
						"",
						"",
						"",
						string(fdata),
						string(v)}
					outputUsers = append(outputUsers, out)
				}
				return nil
			})
		}
		return nil
	})

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(outputUsers)
}

// GetFollowers find all followers detailed info
func GetFollowers(w http.ResponseWriter, r *http.Request) {
	var outputUsers []User
	db, err := bolt.Open(defaultDBName, 0600, nil)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer db.Close()

	db.View(func(tx *bolt.Tx) error {
		f := tx.Bucket([]byte("followers"))
		uf := tx.Bucket([]byte("unfollowers"))
		u := tx.Bucket([]byte("users"))

		if f != nil && u != nil && uf != nil {
			f.ForEach(func(k, v []byte) error {
				udata := u.Get(k)
				ufdata := uf.Get(k)
				if udata != nil && len(udata) > 0 {
					parsed, _ := gabs.ParseJSON(udata)
					jsondata, _ := parsed.ChildrenMap()
					out := User{
						string(k),
						jsondata["login"].Data().(string),
						jsondata["display_name"].Data().(string),
						jsondata["profile_image_url"].Data().(string),
						string(v),
						string(ufdata)}
					outputUsers = append(outputUsers, out)
				} else {
					out := User{
						string(k),
						"",
						"",
						"",
						string(v),
						string(ufdata)}
					outputUsers = append(outputUsers, out)
				}
				return nil
			})
		}
		return nil
	})

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(outputUsers)
}

// GetFollowersID find all followers's ID
func GetFollowersID(w http.ResponseWriter, r *http.Request) {
	var followIDs []int
	db, err := bolt.Open(defaultDBName, 0600, nil)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer db.Close()
	db.View(func(tx *bolt.Tx) error {
		f := tx.Bucket([]byte("followers"))
		if f != nil {
			f.ForEach(func(k, v []byte) error {
				id, err := strconv.Atoi(string(k))
				if err != nil {
					log.Fatal(err)
				}
				followIDs = append(followIDs, id)
				return nil
			})
		}
		return nil
	})

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(followIDs)
}

// GetUnfollowers find all unfollowers
func GetUnfollowers(w http.ResponseWriter, r *http.Request) {
	var unfollowers []Unfollower
	db, err := bolt.Open(defaultDBName, 0600, nil)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer db.Close()
	db.View(func(tx *bolt.Tx) error {
		f := tx.Bucket([]byte("unfollowers"))
		f.ForEach(func(k, v []byte) error {
			u := tx.Bucket([]byte("users"))
			user := u.Get(k)
			parsed, err := gabs.ParseJSON(user)
			if err != nil {
				uf := Unfollower{
					string(k),
					"Unknown",
					"Unknown",
					"Unknown",
					string(v)}
				unfollowers = append(unfollowers, uf)
			} else {
				userdata, _ := parsed.ChildrenMap()
				uf := Unfollower{
					userdata["id"].Data().(string),
					userdata["login"].Data().(string),
					userdata["display_name"].Data().(string),
					userdata["profile_image_url"].Data().(string),
					string(v)}
				unfollowers = append(unfollowers, uf)
			}
			return nil
		})
		return nil
	})
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(unfollowers)
}

// GetUser get specific user
func GetUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	db, err := bolt.Open(defaultDBName, 0600, nil)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer db.Close()

	var user []byte
	db.View(func(tx *bolt.Tx) error {
		u := tx.Bucket([]byte("users"))
		user = u.Get([]byte(id))

		return nil
	})

	if user != nil {
		parsed, _ := gabs.ParseJSON(user)
		userdata, _ := parsed.ChildrenMap()
		uf := Unfollower{
			userdata["id"].Data().(string),
			userdata["login"].Data().(string),
			userdata["display_name"].Data().(string),
			userdata["profile_image_url"].Data().(string),
			""}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(uf)
	} else {
		w.WriteHeader(404)
	}
}
