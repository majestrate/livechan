//
// mod.go
//
// mod events 
//

package main

import (
  "time"
  "fmt"
)

// mod scope
const (
  SCOPE_NIL = iota  // no scope
  SCOPE_POST        // mod does an action on a single post
  SCOPE_CHANNEL     // mod does a channel action
  SCOPE_GLOBAL      // mod does a global action
)

// mod action
const (
  ACTION_NIL = iota         // no action
  ACTION_DELETE_FILE        // delete the file
  ACTION_DELETE_POST        // delete the post and file
  ACTION_DELETE_ALL         // delete all posts and files from this user
  ACTION_BAN                // ban this user
)

type ModEvent struct {
  // the scope of this event
  Scope int
  // the action taken
  Action int
  // the name of the channel this applies to
  // MUST ALWAYS be set
  ChannelName string
  // the post number of the post that was acted upon
  // MUST ALWAYS be set
  PostID int
  // the name of the user that created this action
  UserName string
  // expire time
  Expire int64
}


type Ban struct {
  Offense string
  Date time.Time
  Expiration time.Time
  IpAddr string
}

// forever ban
func (self *Ban) MarkForever() {
  self.Date = time.Now()
  self.Expiration = time.Date(90000, 1, 1, 1, 1, 1, 1, nil) // a long time
}

// cp ban
func (self *Ban) MarkCP() {
  self.MarkForever()
  self.Offense = "CP"
}

// mark ban expires after $duration
func (self *Ban) Expires(bantime time.Duration) {
  self.Date = time.Now()
  self.Expiration = time.Now().Add(bantime)
}

func (self *Ban) String() string {
  return fmt.Sprintf("Reason: %s | Issued: %s | Expires %s ", self.Offense, self.Date.String(), self.Expiration.String())
}
