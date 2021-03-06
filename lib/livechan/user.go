package livechan

import (
  "strconv"
  "time"
  "fmt"
  "log"
  "encoding/gob"
)


var USER_PROP_SCOPE = string("modScope")
var USER_PROP_LEVEL = string("modLevel")

/* Registered users can moderate, own channels, etc. */
type User struct {
  Name string
  //Password string
  Created time.Time
  //Identifiers string // JSON
  Attributes map[string]string

  // user's current ip address
  IpAddr string
  Session string
  SolvedCaptcha bool
}

func CreateUser() *User {
  return &User{Attributes: make(map[string]string)}
}

func init() {
  // register user with gob for sessions
  gob.Register(&User{})
}

// generate channel property name
// conveniece
func chanPropName(chanName string, propName string) string {
  return fmt.Sprintf("channel.%s.%s", chanName, propName)
}

// generate global property name
// conveniece
func globalPropName(propName string) string {
  return fmt.Sprintf("global.%s", propName)
}


// mark this user as having solved a captcha
func (user *User) MarkSolvedCaptcha() {
  user.SolvedCaptcha = true
  log.Println(user.IpAddr, "solved captcha")
}

// attempt login to the moderation system
// return true on success otherwise false
func (user *User) Login(username, password string) bool {
  user.Name = username
  if storage.checkModLogin(user.Name, password) {
    // refresh this user's information
    return user.Refresh()
  }
  return false
}

// return attribute as string
func (user *User) Get(name string) string {
  attr , ok := user.Attributes[name]
  if ok {
    return attr
  }
  return ""
}

// do we require this user to follow the cooldown rules?
// return true if so otherwise false
func (user *User) RequireCooldown() bool {
  return ! (user.IsAdmin() || user.GetInt(globalPropName("bypassCooldown")) == 1 )
}

// return attribute as int
// return -1 on error
func (user *User) GetInt(name string) int {
  attr, ok := user.Attributes[name]
  if ok {
    val, err := strconv.Atoi(attr)
    if err == nil {
      return val
    }
  }
  return -1
}


// refresh the user's information from the backend
// return true on success otherwise false
func (user *User) Refresh() bool {
  // get our user's attributes
  user.Attributes = storage.getModAttributes(user.Name)
  // XXX: do more stuff if needed
  return true
}

// save and remember attributes
func (user *User) Store() {
  for attr := range(user.Attributes) {
    storage.setModAttribute(user.Name, attr, user.Attributes[attr])
  }
}

// grant this user admin
func (user *User) GrantAdmin() {
  user.Attributes["admin"] = "1"
}

// are we admin ?
func (user *User) IsAdmin() bool {
  return user.GetInt("admin") == 1
}

// return true if we can do action on this scope
func (user *User) PermitAction(channelName string, scope, action int) bool {

  // admin can do whatever
  if user.IsAdmin() {
    return true
  }
  switch(scope) {
  case SCOPE_POST:
  case SCOPE_CHANNEL:
    return user.GetInt(chanPropName(channelName, USER_PROP_SCOPE)) >= scope && user.GetInt(chanPropName(channelName, USER_PROP_LEVEL)) >= action
  case SCOPE_GLOBAL:
    return user.GetInt(globalPropName(USER_PROP_SCOPE)) >= scope && user.GetInt(globalPropName(USER_PROP_LEVEL)) >= action
  default:
    break
  }
  return false
}

func (user *User) Moderate(scope, action int, channelName string, postID int, expire int64, reason string) bool {
  // can we moderate?
  if user.PermitAction(channelName, scope, action) {
    // yes we can!
    // send the event to the event hub
    h.mod <- ModEvent{scope, action, channelName, postID, user.Name, expire, reason}
    return true
  }
  // no we cannot do this action
  log.Println("invalid mod action attempt by", user.Name, "channel=", channelName, "action=", action, "scope=", scope, "postID=", postID)
  return false
}

  
