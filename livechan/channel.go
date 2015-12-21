package livechan

import (
  "bytes"
  "time"
  "strconv"
  "strings"
  "log"
)

const (
  restr_none = 0       // No restriction
  restr_rate = 1       // Rate limited (4 seconds)
  restr_captcha = 2    // Captcha sessions (24 hours)
  restr_nofiles = 4    // No uploads
  restr_registered = 8 // Users must be registered
  gen_country = 1
  gen_id = 2
)

type ChannelInfo struct {
  Name string
  Restrictions uint64
  Generated uint64
  Options string // JSON
}

type Owner struct {
  User string
  Channel string
  Permissions uint64
}

type Channel struct {
  // connections for this channel
  Connections map[*Connection]time.Time
  // converstations in this channel
  // TODO: use this
  Convos []string
  Scrollback uint64
  Name string
  // chan for recving incoming chats to send to this channel
  Send chan Chat
}

func NewChannel(name string) *Channel {
  chnl := new(Channel)
  chnl.Name = strings.ToLower(name)
  chnl.Send = make(chan Chat, 20)
  var fallbackScrollback uint64
  fallbackScrollback = 50
  // if we have set a scrollback amount in our config set it here
  if cfg.Has("scrollback") {
    var err error
    chnl.Scrollback, err = strconv.ParseUint(cfg["scrollback"], 10, 64)
    if err != nil {
      chnl.Scrollback = fallbackScrollback
    }
  } else {
    chnl.Scrollback = fallbackScrollback
  }
  chnl.Convos = make([]string, 10)
  chnl.Connections = make(map[*Connection]time.Time)
  return chnl
}

// broadcast an OutChat to everyone
func (self *Channel) BroadcastOutChat(chat *OutChat) {
  var buff bytes.Buffer
  chat.createJSON(&buff)
  data := buff.Bytes()
  for con := range self.Connections {
    con.send <- data
  }
  buff.Reset()
}

// run channel mainloop
func (self *Channel) Run() {
  for {

    // we got a chat!
    select {
    case chat := <- self.Send:
      // register the chat with the channel
      // sets post number etc
      self.RegisterWithChannel(&chat)
      // broadcast it
      var ch = chat.toOutChat()
      self.BroadcastOutChat(&ch)
    }
  }
}

// register this post as being in this channel
// sets post number
// saves the post
func (self *Channel) RegisterWithChannel(chat *Chat) {
  chat.Count = storage.getCount(self.Name) + 1
  storage.insertChat(self, chat)
}

// return true if this connection is allowed to post
// checks for rate limits
func (self *Channel) ConnectionCanPost(con *Connection) bool {
  // get last post time
  t := self.Connections[con]
  // are we good with cooldown?
  if uint64(time.Now().Sub(t).Seconds()) < self.GetCooldown() {
    // nope
    return false
  }
  
  // TODO: other checks
  
  return true
}

// get post cooldown time
func (self *Channel) GetCooldown() uint64 {
  var cooldown uint64
  cooldown = 4
  // TODO: use channel specific settings
  if cfg.Has("cooldown") {
    _cooldown, err := strconv.ParseUint(cfg["cooldown"], 10, 64)
    if err == nil {
      cooldown = _cooldown
    }
  }
  return cooldown
}

func (self *Channel) RemoveConnection(conn *Connection) {
  if _, ok := self.Connections[conn]; ok {
    // remove them from the list of connections
    delete(self.Connections, conn)
  }
}

func (self *Channel) Part(conn *Connection) {
  // remove our connection from the list
  self.RemoveConnection(conn)
  // anounce user part    
  var chat OutChat
  chat.UserCount = len(self.Connections)
  self.BroadcastOutChat(&chat)
}

func (self *Channel) Join(conn *Connection) {
  // connection joined, add it to the list
  self.Connections[conn] = time.Now()
  // anounce new user join
  var chat OutChat
  chat.UserCount = len(self.Connections)
  self.BroadcastOutChat(&chat)
}

// record that a post was made
func (self *Channel) ConnectionPosted(conn *Connection) {
  // record post event
  now := time.Now()
  // if this user needs to follow the cooldown rules do the cooldown
  if conn.user.RequireCooldown() {
    self.Connections[conn] = now
  }
  // log it
  log.Println(conn.ipAddr, "posted at", now.Unix())
}
