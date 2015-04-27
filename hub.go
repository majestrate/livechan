package main

import (
  "bytes"
  "time"
  "fmt"
  //"io"
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
            log.Println("captcha solved")
          }
        }
      }
      // check for new connection events
    case con := <-h.register:
      // get channel
      chnl := h.getChannel(con.channelName)
      // put user presence
      chnl.Connections[con] = time.Unix(0,0)
      // send scollback
      ch := storage.getChats(con.channelName, "General", chnl.Scrollback)
      createJSONs(ch, con.send)
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
        var buff bytes.Buffer
        chat.createJSON(&buff)
        m.conn.send <- buff.Bytes()
        buff.Reset()
      } else {
        chnl := h.channels[chName]
        // can we post?
        conn := m.conn
        if chnl.ConnectionCanPost(conn) {
          // yes
          // set last posted to now
          chnl.Connections[conn] = time.Now()
          // send the result down the channel's recv chan
          chnl.Send <- m.chat
        }
      }
    }
  }
}

