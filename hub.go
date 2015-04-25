package main

import (
  "time"
  "fmt"
  "log"
)


// raw json message
type Message struct {
  data []byte
  conn *Connection
}

// hub main type
type Hub struct {

  // channel specific broadcast 
  channels map[string]*Channel

  // regular channel message events
  broadcast chan Message

  // moderation based events
  mod chan ModEvent

  // captcha events
  captcha chan string

  // register a new connection
  register chan *Connection

  // unregister a connection
  unregister chan *Connection
}

// todo: shouldn't this be made in main?
var h = Hub {
  broadcast: make(chan Message),
  mod: make(chan ModEvent),
  captcha: make(chan string),
  register: make(chan *Connection),
  unregister: make(chan *Connection),
  channels: make(map[string]*Channel),
}

// hub mainloop
// TODO: locking?
func (h *Hub) run() {
  for {
    select {

      // check for mod event
    case ev := <-h.mod:
      log.Println("Got Mod event: ", fmt.Sprintf("%q", ev))
      // execute the mod event so it doesn't block
      go storage.ProcessModEvent(ev.Scope, ev.Action, ev.ChannelName, ev.PostID, ev.Expire)
      
      // check for captcha solved events
    case sid := <-h.captcha:
      // find the connections with this session ID
      for _, ch := range(h.channels) {
        for conn, _ := range(ch.Connections) {
          if conn.user.Session == sid {
            // mark underlying user object as solved captcha
            conn.user.MarkSolvedCaptcha()
          }
        }
      }
      // check for new connection events
    case con := <-h.register:
      // channel has no users?
      if (h.channels[con.channelName] == nil) {
        // allocate channel
        h.channels[con.channelName] = NewChannel(con.channelName)
      }
      chnl := h.channels[con.channelName]
      // put user presence
      chnl.Connections[con] = time.Unix(0,0)
      // send scollback
      con.send <- createJSONs(storage.getChats(con.channelName, "General", chnl.Scrollback), con)

      // call channel OnJoin
      chnl.OnJoin(con)
      
      // unregister connection
    case con := <-h.unregister:
      chname := con.channelName
      // check for existing presence
      chnl, ok := h.channels[chname]
      if ok {
        chnl.OnPart(con)
      } else {
        log.Println("no such channel to unregister user from", chname)
      }
    case m := <-h.broadcast:
      chName := m.conn.channelName
      ipaddr := ExtractIpv4(m.conn.ipAddr)
      // check for banned
      if storage.isGlobalBanned(ipaddr) {
        // tell them they are banned
        var chat OutChat
        chat.Error = "You have been banned from Livechan: "
        chat.Error += storage.getGlobalBanReason(ipaddr)
        // send them the ban notice 
        m.conn.send <- chat.createJSON()
      } else {
        h.channels[chName].OnBroadcast(m)
      }
    }
  }
}

