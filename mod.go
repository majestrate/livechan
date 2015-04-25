//
// mod.go
//
// mod events 
//

package main

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

