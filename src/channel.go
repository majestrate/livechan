package main

import (
  "time"
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

type Channel interface {
  Name() string
  // channel for others to send chat to us
  Chan() chan ChannelChat
  // register a connection with this channel and announce join
  Join(conn Connection)
  // deregister a connection with this channel and announce part
  Part(conn Connection)
  // check if a connection can post right now
  CanPost(conn Connection) bool
  // run the channel mainloop
  Run()
  // end the channel's existence
  // clean up anything
  End()
}

type liveChannel struct {
  // connection -> cooldown
  connections map[Connection]time.Time
  name string
  // chan for recving incoming chats to send to this channel
  chnlChatChnl chan ChannelChat
  // configuration for this channel
  config ChannelConfig
}

// get our channel
func (self liveChannel) Chan() chan ChannelChat {
  return self.chnlChatChnl
}

// broadcast an OutChat to everyone
func (self liveChannel) broadcastOutChat(chat OutChat) {
  for con := range self.connections {
    con.Chan() <- chat
  }
}

func (self liveChannel) End() {
  // part everyone from the channel
  for conn, _ := range(self.connections) {
    self.Part(conn)
  }
  // close the chan for channel chat
  close(self.chnlChatChnl)
}

// run channel mainloop
func (self liveChannel) Run() {
  chnl := self.Chan()
  for {

    // we got a chat!
    select {
    case chnlchat := <- self.chnlChatChnl:
      // can we post?
      if self.CanPost(chnlchat.conn) {
        // yas
        // register the chat
        self.posted(chnlchat)
        // broadcast it
        c := chnlchat.chat
        ch := c.toOutChat()
        self.broadcastOutChat(ch)
      }
    }
  }
}

// return true if this connection is allowed to post
// checks for rate limits
func (self liveChannel) CanPost(con Connection) bool {
  // get last post time
  t := self.connections[con]
  // are we good with cooldown?
  if int64(time.Now().Sub(t).Seconds()) < self.getCooldown() {
    // nope
    return false
  }
  
  // TODO: other checks
  
  return true
}

// get post cooldown time
func (self liveChannel) getCooldown() int64 {
  // TODO: use channel specific settings
  return self.config.GetInt("cooldown", 4)
}

func (self liveChannel) removeConnection(conn Connection) {
  if _, ok := self.connections[conn]; ok {
    // remove them from the list of connections
    delete(self.connections, conn)
  }
}

func (self liveChannel) Part(conn Connection) {
  // remove our connection from the list
  self.removeConnection(conn)
  // anounce user part    
  var chat OutChat
  chat.UserCount = len(self.connections)
  self.broadcastOutChat(chat)
  // close the connection for them
  conn.Close()
}

func (self liveChannel) Join(conn Connection) {
  // connection joined, add it to the list
  self.connections[conn] = time.Now()
  // anounce new user join
  var chat OutChat
  chat.UserCount = len(self.connections)
  self.broadcastOutChat(chat)
}

// record that a post was made
func (self liveChannel) posted(chnlChat ChannelChat) {
  // record post event
  now := time.Now()
  conn := chnlChat.conn
  // if this user needs to follow the cooldown rules do the cooldown
  if conn.RequireCooldown() {
    self.connections[conn] = now
  }
  // log it
  log.Println(conn.Addr(), "posted at", now.Unix())
}

func (self liveChannel) Name() string {
  return self.name
}
