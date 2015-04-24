package main

import (
  "time"
)


// raw json message
type Message struct {
  data []byte
  conn *Connection
}

// hub main type
type Hub struct {

  // channel specific broadcast 
  channels map[string]map[*Connection]time.Time

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
  channels: make(map[string]map[*Connection]time.Time),
}

// hub mainloop
// TODO: locking?
func (h *Hub) run() {
  for {
    select {

      // check for mod event
    case ev := <-h.mod:
      
      
      // check for captcha solved events
    case sid := <-h.captcha:
      // find the connections with this session ID
      for _, ch := range(h.channels) {
        for conn, _ := range(ch) {
          if conn.user.Session == sid {
            // mark underlying user object as solved captcha
            conn.user.MarkSolvedCaptcha()
          }
        }
      }
      // check for new connection events
    case c := <-h.register:
      // channel has no users?
      if (h.channels[c.channelName] == nil) {
        // allocate channel
        h.channels[c.channelName] = make(map[*Connection]time.Time)
      }
      // put user presence
      h.channels[c.channelName][c] = time.Unix(0,0)
      // send the last 50 chat messages of scrollback
      // TODO: make scrollback variable?
      c.send <- createJSONs(storage.getChats(c.channelName, "General", 50), c)
      
      // anounce new user join
      var chat OutChat
      chat.UserCount = len(h.channels[c.channelName])
      jsondata := chat.createJSON()
      // send to everyone in this channel
      for ch := range h.channels[c.channelName] {
          ch.send <- jsondata
      }
      
      // unregister connection
    case c := <-h.unregister:
      // check for existing presence
      if _, ok := h.channels[c.channelName][c]; ok {
        delete(h.channels[c.channelName], c)
        close(c.send)
        // anounce user part
        var chat OutChat
        chat.UserCount = len(h.channels[c.channelName])
        jsondata := chat.createJSON()
        // tell everyone in channel the user count  decremented
        for ch := range h.channels[c.channelName] {
            ch.send <- jsondata
        }
      } else {} // do nothing if presence does not exist
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
        // if they aren't banned create the chat message
        chat := createChat(m.data, m.conn)
        // check if we can broadcast to the channel
        // potentially check for +m
        if (chat.canBroadcast(m.conn)) {
          for c := range h.channels[chName] {
            // for each connection send a chat message
            select {
              // send it 
            case c.send <- chat.createJSON(c):
              // if we can't send it unregister the chat
            default:
              // TODO is this okay?
              h.unregister <- c
            }
          }
          storage.insertChat(chName, *chat)
        } else {} // TODO: should we really do nothing when the channel can't broadcast?
      }
    }
  }
}

