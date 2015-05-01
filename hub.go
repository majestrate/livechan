package main

import (
  "log"
)


// chat message
type Message struct {
  chat Chat
  conn *Connection
}

// hub main type
type Hub struct {

  // channel specific broadcast 
  channels map[string]*Channel

  // incoming regular channel message events
  broadcast chan Message

  // moderation based events
  mod chan ModEvent

  // captcha events
  captcha chan *User

  // register a new connection
  register chan *Connection

  // unregister a connection
  unregister chan *Connection
}

// todo: shouldn't this be made in main?
var h = Hub {
  broadcast: make(chan Message),
  mod: make(chan ModEvent),
  captcha: make(chan *User),
  register: make(chan *Connection),
  unregister: make(chan *Connection),
  channels: make(map[string]*Channel),
}

func (h *Hub) RemoveChannel(chnl *Channel) {
  // remove it
  log.Println("remove channel", chnl.Name)
  delete(h.channels, chnl.Name)
}

func (h *Hub) getChannel(chname string) *Channel {
  if (h.channels[chname] == nil) {
    // allocate channel
    ch := NewChannel(chname)
    // put it into the hub
    h.channels[chname] = ch
        // run the channel pumper
    go ch.Run()
  }
  return h.channels[chname]
}

// hub mainloop
// TODO: locking?
func (h *Hub) run() {
  for {
    select {

      // check for mod event
    case ev := <-h.mod:
      // execute the mod event so it doesn't block
      go storage.ProcessModEvent(ev)
      
      // check for captcha solved events
    case u := <-h.captcha:
      // find user that matches our user via IP
      // mark them as solved for all channels
      for chName := range(h.channels) {
        chnl := h.channels[chName]
        for conn := range(chnl.Connections) {
          // we should use sessions here instead :x
          if u.IpAddr == conn.user.IpAddr {
            conn.user.MarkSolvedCaptcha()
          }
        }
      }
      
      // check for new connection events
    case con := <-h.register:
      // get channel
      chnl := h.getChannel(con.channelName)
      // join the channel
      chnl.Join(con)
      // send scollback
      ch := storage.getChats(con.channelName, "General", chnl.Scrollback)
      createJSONs(ch, con.send)
      
      // unregister connection
      // we assume the connection's websocket is already closed
    case con := <-h.unregister:
      chname := con.channelName
      // check for existing presence
      chnl, ok := h.channels[chname]
      if ok {
        // handle removal of connection from channel
        // tell everyone in channel they left
        chnl.Part(con)
      } else {
        log.Println(con.ipAddr, "no such channel to unregister user from", chname)
      }
    case m := <-h.broadcast:
      conn := m.conn
      chName := conn.channelName
      // this shouldn't create a new channel but do that just in case (tm)
      chnl := h.getChannel(chName)
      if chnl.ConnectionCanPost(conn) {
        // yes
        // set last posted to now
        chnl.ConnectionPosted(conn)
        // send the result down the channel's recv chan
        chnl.Send <- m.chat
      }
    }
  }
}

