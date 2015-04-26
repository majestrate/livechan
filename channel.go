package main

import (
  "bytes"
  "time"
  "strconv"
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

type Channel struct {
  // connections for this channel
  Connections map[*Connection]time.Time
  // converstations in this channel
  // TODO: use this
  Convos []string
  Scrollback uint64
  Name string
}

func NewChannel(name string) *Channel {
  chnl := new(Channel)
  chnl.Name = name
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

func (self *Channel) OnBroadcast(msg *Message) {
  // if they aren't banned create the chat message
  var chat Chat
  createChat(msg.reader, msg.conn, &chat)
  // check if we can broadcast to the channel
  // potentially check for +m
  if (chat.canBroadcast(msg.conn)) {
    for con := range self.Connections {
      var buff bytes.Buffer
      chat.createJSON(con, &buff)
      // for each connection send a chat message
      select {
        // send it 
      case con.send <- buff.Bytes():
        // if we can't send it unregister the chat
      default:
        // TODO is this okay?
        h.unregister <- con
      }
    }
    storage.insertChat(self, chat)
  } else {} // TODO: should we really do nothing when the channel can't broadcast?
}

func (self *Channel) OnPart(conn *Connection) {
  if _, ok := self.Connections[conn]; ok {
    delete(self.Connections, conn)
    close(conn.send)
    // anounce user part
    
    var chat OutChat
    var buff bytes.Buffer
    chat.UserCount = len(self.Connections)
    chat.createJSON(&buff)
    // tell everyone in channel the user count  decremented
    for c := range self.Connections {
      c.send <- buff.Bytes()
    }
  }
}

func (self *Channel) OnJoin(conn *Connection) {
  // anounce new user join
  var chat OutChat
  var buff bytes.Buffer
  chat.UserCount = len(self.Connections)
  chat.createJSON(&buff)
  // send to everyone in this channel
  for ch := range self.Connections {
    ch.send <- buff.Bytes()
  }
}
