package main

import (
  "time"
)

type Message struct {
  data []byte
  conn *Connection
}

type Hub struct {
  channels map[string]map[*Connection]time.Time
  broadcast chan Message
  mod chan *User
  captcha chan string
  register chan *Connection
  unregister chan *Connection
}

var h = Hub {
  broadcast: make(chan Message),
  mod: make(chan *User),
  captcha: make(chan string),
  register: make(chan *Connection),
  unregister: make(chan *Connection),
  channels: make(map[string]map[*Connection]time.Time),
}

func (h *Hub) run() {
  for {
    select {
    case sid := <-h.captcha:
      for _, ch := range(h.channels) {
        for conn, _ := range(ch) {
          if conn.user.Session == sid {
            conn.user.MarkSolvedCaptcha()
          }
        }
      }
    case c := <-h.register:
      if (h.channels[c.channelName] == nil) {
        h.channels[c.channelName] = make(map[*Connection]time.Time)
      }
      h.channels[c.channelName][c] = time.Unix(0,0)
      c.send <- createJSONs(storage.getChats(c.channelName, "General", 50), c)
      
      // anounce new user join
      var chat OutChat
      chat.UserCount = len(h.channels[c.channelName])
      jsondata := chat.createJSON()
      for ch := range h.channels[c.channelName] {
          ch.send <- jsondata
      }
      
    case c := <-h.unregister:
      if _, ok := h.channels[c.channelName][c]; ok {
        delete(h.channels[c.channelName], c)
        close(c.send)
        // anounce user part
        var chat OutChat
        chat.UserCount = len(h.channels[c.channelName])
        jsondata := chat.createJSON()
        for ch := range h.channels[c.channelName] {
            ch.send <- jsondata
        }
      }
    case m := <-h.broadcast:
      chName := m.conn.channelName
      ipaddr := m.conn.ipAddr
      // check for banned
      if storage.isGlobalBanned(ipaddr) {
        var chat OutChat
        chat.Error = "You have been banned from Livechan"
        m.conn.send <- chat.createJSON()
      } else {
        var chat = createChat(m.data, m.conn);
        if (chat.canBroadcast(m.conn)) {
          for c := range h.channels[chName] {
            select {
            case c.send <- chat.createJSON(c):
            default:
              close(c.send)
              delete(h.channels[chName], c)
            }
          }
        storage.insertChat(chName, *chat)
      }
      }
    }
  }
}

