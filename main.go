package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/boltdb/bolt"
)

func main() {
	fmt.Printf("[SYS] Welcome to TUT, Twitch Unfollow Tacker, v%s\n", version)
	conf := initialize()
	go backendServer(conf.serverPort)

	fmt.Printf("[SYS] Starting... \n")
	fmt.Printf("[SYS] Using %+v \n", conf)
	for {
		// fmt.Printf("[SYS] Update Followers Snippet... \n")
		monitor(conf)

		nextUpdate := time.Now().Add(time.Duration(conf.updateInterval) * time.Minute)
		// fmt.Printf("[SYS] Next Update scheduled at [%s]\n", nextUpdate)
		for nextUpdate.Sub(time.Now()).Seconds() > 0 {
			done := updateUsers(conf)
			if done {
				time.Sleep(nextUpdate.Sub(time.Now()))
			}
		}
	}
}

func initialize() config {
	var clientID string
	var oauth string
	var username string
	var userID string
	var serverPort string
	var updateInterval int

	db, err := bolt.Open(defaultDBName, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Try to use bucket "config" and find clientID
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b == nil {
			newb, err := tx.CreateBucket([]byte("config"))
			if err != nil {
				return err
			}
			err = newb.Put([]byte("clientID"), []byte(defaultClientID))
			if err != nil {
				return err
			}
			clientID = defaultClientID
			updateInterval = defaultUpdateInterval
		} else {
			clientID = string(b.Get([]byte("clientID")))
			oauth = string(b.Get([]byte("oauth")))
			updateInterval, _ = strconv.Atoi(string(b.Get([]byte("updateInterval"))))
		}
		return nil
	})

	// Ask user whether to use saved clientID or new clientID
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Simply Enter to use ClientID [%s] or Enter your ClientID: ", clientID)
	scanner.Scan()
	inputClinetID := scanner.Text()

	// Update clientID if there is userinput
	if len(inputClinetID) > 0 {
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("config"))
			err = b.Put([]byte("clientID"), []byte(inputClinetID))
			if err != nil {
				return err
			}
			return nil
		})
		clientID = inputClinetID
	}

	// Ask user whether to use saved AccessToken or new AccessToken
	fmt.Printf("Simply Enter to use Access token [%s] or Enter your New Access token [Optional]: ", oauth)
	scanner.Scan()
	inputOAuth := scanner.Text()

	// Update clientID if there is userinput
	if len(inputOAuth) > 0 {
		inputOAuth = strings.Replace(inputOAuth, "oauth:", "", 1)
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("config"))
			err = b.Put([]byte("oauth"), []byte(inputOAuth))
			if err != nil {
				return err
			}
			return nil
		})
		oauth = inputOAuth
	}

	// Try to get userID from username
	for len(username) == 0 {
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("config"))
			username = string(b.Get([]byte("username")))
			userID = string(b.Get([]byte("userID")))
			return nil
		})

		// Ask user for username
		if username == "" || userID == "" {
			fmt.Printf("Enter Twitch Username to track: ")
		} else {
			fmt.Printf("Simply Enter to use Username [%s] or Enter your Username: ", username)
		}
		scanner.Scan()
		inputUsername := scanner.Text()

		// Update inputUsername if there is user input
		if len(inputUsername) != 0 {
			username = string([]byte(inputUsername))
			db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("config"))
				err = b.Put([]byte("username"), []byte(username))
				if err != nil {
					return err
				}
			getUserID:
				result, err := getUserIDFromTwitch(username, clientID, oauth)
				if result.statusCode != 200 && result.limitRemaining == 0 {
					waitTime := time.Unix(result.limtResetTime, 0).Sub(time.Now())
					// fmt.Printf("[SYS] Waiting for API Limit Reset (%s)...\n", waitTime)
					time.Sleep(waitTime)
					// fmt.Println("[SYS] API Limit Reset Done...")
					goto getUserID
				}
				if err != nil {
					log.Fatal(err)
				}
				err = b.Put([]byte("userID"), []byte(result.response["id"]))
				if err != nil {
					return err
				}
				userID = result.response["id"]
				return nil
			})
		}
	}

	// Ask user whether to use saved OAuth or new OAuth
	fmt.Printf("Simply Enter to use update interval [%d] minutes or Enter your update interval: ", updateInterval)
	scanner.Scan()
	inputUpdateInterval := scanner.Text()

	// Update clientID if there is userinput
	if len(inputUpdateInterval) > 0 {
		updateInterval, err = strconv.Atoi(inputUpdateInterval)
		if err != nil {
			log.Fatal("Please enter a valid number for update interval")
		}
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("config"))
			err = b.Put([]byte("updateInterval"), []byte(inputUpdateInterval))
			if err != nil {
				return err
			}
			return nil
		})
	}

	// Try to get serverPort
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		port := b.Get([]byte("serverPort"))
		if port != nil {
			serverPort = string(port)
		} else {
			b.Put([]byte("serverPort"), []byte(defaultPort))
			serverPort = defaultPort
		}
		return nil
	})

	// Ask user whether to use saved server port or enter new server port
	fmt.Printf("Simply Enter to use server port [%s] or Enter your server port: ", serverPort)
	scanner.Scan()
	inputServerPort := scanner.Text()

	// Update clientID if there is userinput
	if len(inputServerPort) > 0 {
		_, isInt := strconv.Atoi(inputServerPort)
		if isInt != nil {
			log.Fatal("Please enter valid port")
		}
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("config"))
			err = b.Put([]byte("serverPort"), []byte(inputServerPort))
			if err != nil {
				return err
			}
			return nil
		})
		serverPort = inputServerPort
	}

	// Try to create follower bucket and userID bucket inside it
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("followers"))
		if b == nil {
			_, err := tx.CreateBucket([]byte("followers"))
			if err != nil {
				return err
			}
		}
		return nil
	})

	// Try to create unfollower and userID bucket
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("unfollowers"))
		if b == nil {
			_, err := tx.CreateBucket([]byte("unfollowers"))
			if err != nil {
				return err
			}
		}
		return nil
	})

	// Try to create user bucket
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			_, err := tx.CreateBucket([]byte("users"))
			if err != nil {
				return err
			}
		}
		return nil
	})

	return config{clientID, oauth, username, userID, serverPort, updateInterval}
}

func monitor(c config) {
	// Get all followers and unfollowers from previous snippet
	db, err := bolt.Open(defaultDBName, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	followMap := make(map[string]string)
	unfollowMap := make(map[string]string)
	db.View(func(tx *bolt.Tx) error {
		f := tx.Bucket([]byte("followers"))
		f.ForEach(func(k, v []byte) error {
			followMap[string(k)] = string(v)
			return nil
		})

		uf := tx.Bucket([]byte("unfollowers"))
		uf.ForEach(func(k, v []byte) error {
			unfollowMap[string(k)] = string(v)
			return nil
		})
		return nil
	})
	db.Close()

	// Get next page if there is any
	var page string
	for {
		var toAdd []follower
		result, out, _ := getFollowersFromTwitch(c.userID, page, c.clientID, c.oauth)

		if result.statusCode != 200 && result.limitRemaining == 0 {
			waitTime := time.Unix(result.limtResetTime, 0).Sub(time.Now())
			// fmt.Printf("[SYS] Waiting for API Limit Reset (%s)...\n", waitTime)
			time.Sleep(waitTime)
			// fmt.Println("[SYS] API Limit Reset Done...")
			continue
		}

		if len(out) == 0 {
			break
		}

		page = result.response["next"]

		// Filter out followers
		for _, follower := range out {
			_, exist := followMap[follower.uid]
			if exist {
				delete(followMap, follower.uid)
			} else {
				_, refollow := unfollowMap[follower.uid]

				if refollow {
					db, err = bolt.Open(defaultDBName, 0600, nil)
					if err != nil {
						log.Fatal(err)
					}

					// Try to find user data in user bucket
					var displayname, login string
					db.View(func(tx *bolt.Tx) error {
						u := tx.Bucket([]byte("users"))
						userdata := u.Get([]byte(follower.uid))
						if userdata != nil {
							parsed, _ := gabs.ParseJSON([]byte(userdata))
							user, _ := parsed.ChildrenMap()
							displayname, _ = user["display_name"].Data().(string)
							login, _ = user["login"].Data().(string)
						}
						return nil
					})
					db.Close()

					// If user data is not presetned in user bucket, we querry twitch API
					if displayname == "" && login == "" {
					getUserNameInRefollow:
						result, _ := getUserNameFromTwitch(follower.uid, c.clientID, c.oauth)
						if result.statusCode != 200 && result.limitRemaining == 0 {
							waitTime := time.Unix(result.limtResetTime, 0).Sub(time.Now())
							// fmt.Printf("[SYS] Waiting for API Limit Reset (%s)...\n", waitTime)
							time.Sleep(waitTime)
							// fmt.Println("[SYS] API Limit Reset Done...")
							goto getUserNameInRefollow
						}
						displayname = result.response["displayname"]
						login = result.response["login"]
					}

					fmt.Printf("[INFO][RE-FOLLOW] %s (%s) [%s] Followed: %s\n", displayname, login, follower.uid, follower.followedAt)
				} else {
					fmt.Printf("[INFO][FOLLOW] UID: %s Followed: %s\n", follower.uid, follower.followedAt)
				}
				toAdd = append(toAdd, follower)
			}
		}

		// Commit changes
		db, err = bolt.Open(defaultDBName, 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		db.Update(func(tx *bolt.Tx) error {
			f := tx.Bucket([]byte("followers"))

			for _, v := range toAdd {
				err := f.Put([]byte(v.uid), []byte(v.followedAt))
				if err != nil {
					return err
				}
			}

			return nil
		})
		db.Close()
	}

	// Found unfollower
	for k, v := range followMap {
	getUserNameInUnfollow:
		result, _ := getUserFromTwitch(k, c.clientID, c.oauth)
		if result.statusCode != 200 && result.limitRemaining == 0 {
			waitTime := time.Unix(result.limtResetTime, 0).Sub(time.Now())
			// fmt.Printf("[SYS] Waiting for API Limit Reset (%s)...\n", waitTime)
			time.Sleep(waitTime)
			// fmt.Println("[SYS] API Limit Reset Done...")
			goto getUserNameInUnfollow
		}

		parsed, err := gabs.ParseJSON([]byte(result.response["user"]))
		if err != nil {
			fmt.Printf("[INFO][UNFOLLOW / ID Not exist] [%s], Followed: %s\n", k, v)
		} else {
			userdata, _ := parsed.ChildrenMap()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("[INFO][UNFOLLOW] %s (%s) [%s], Followed: %s\n", userdata["display_name"].Data().(string), userdata["login"].Data().(string), k, v)
		}

		db, err = bolt.Open(defaultDBName, 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		db.Update(func(tx *bolt.Tx) error {
			// remove the unfollower from followers bucket
			f := tx.Bucket([]byte("followers"))
			err := f.Delete([]byte(k))
			if err != nil {
				return err
			}

			// Add the unfollower to the unfollowers bucket
			uf := tx.Bucket([]byte("unfollowers"))
			err = uf.Put([]byte(k), []byte(time.Now().UTC().Format(time.RFC3339)))
			if err != nil {
				return err
			}

			// Add detailed unfollowed user info into users bucket
			u := tx.Bucket([]byte("users"))
			err = u.Put([]byte(k), []byte(result.response["user"]))
			if err != nil {
				return err
			}
			return nil
		})
		db.Close()
	}
}

func updateUsers(c config) bool {
	alldone := true
	db, err := bolt.Open(defaultDBName, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	var waitTime time.Duration
	db.Update(func(tx *bolt.Tx) error {
		f := tx.Bucket([]byte("followers"))
		u := tx.Bucket([]byte("users"))

		cur := f.Cursor()
		for k, _ := cur.First(); k != nil; k, _ = cur.Next() {
			data := u.Get(k)
			if data == nil {
				result, _ := getUserFromTwitch(string(k), c.clientID, c.oauth)
				if result.statusCode != 200 && result.limitRemaining == 0 {
					waitTime = time.Unix(result.limtResetTime, 0).Sub(time.Now())
					alldone = false
					break
				}
				err := u.Put([]byte(k), []byte(result.response["user"]))
				if err != nil {
					return err
				}
			}
		}
		// followerCount := 0
		// userCount := 0
		// f.ForEach(func(_, _ []byte) error {
		// 	followerCount++
		// 	return nil
		// })
		// u.ForEach(func(_, _ []byte) error {
		// 	userCount++
		// 	return nil
		// })
		// fmt.Printf("[SYS][%s] Checked / Updated some users info...[%d/%d]\n", time.Now().Local(), userCount, followerCount)
		return nil
	})
	db.Close()

	if waitTime.Seconds() > 0 {
		time.Sleep(waitTime)
	}

	return alldone
}
