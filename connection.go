package main

import (
  "bytes"
  "github.com/gorilla/websocket"
  "log"
  //"strings"
  "time"
)

const (
  writeWait = 10 * time.Second     // Write timeout.
  pongWait = 60 * time.Second      // Read timeout.
  pingPeriod = (pongWait * 9) / 10 // How frequently to ping the clients.
  maxMessageSize = 1024 * 1024     // Maximum size of a message.
)

/* A Connection will maintain all data pertinent to an active
   websocket connection. */
type Connection struct {
  ws *websocket.Conn
  send chan []byte
  channelName string
  ipAddr string
  user *User // user info
}

// explicit close
func (c *Connection) Close() {
  h.unregister <- c
  c.ws.Close()
}

/* @brief Read until there is an error. */
func (c *Connection) reader() {
  c.ws.SetReadLimit(maxMessageSize)
  c.ws.SetReadDeadline(time.Now().Add(pongWait))
  c.ws.SetPongHandler(func(string) error {
    c.ws.SetReadDeadline(time.Now().Add(pongWait))
    return nil
  })
  for {
    _, d, err := c.ws.ReadMessage()
    if err != nil {
      break
    }
    // did we solve the captcha?
    if c.user.SolvedCaptcha {
      // ya, create chat
      go createChat(d, c)
    } else {
      log.Println(c.user.IpAddr, "needs to solve captcha")
      // nah, send captcha challenge
      var chat OutChat
      chat.Notify = "Please fill in the captcha"
      var buff bytes.Buffer
      chat.createJSON(&buff)
      c.send <- buff.Bytes()
    }
    
  }
}

/* @brief Sends data to the connection.
 *
 * @param mt The type of message.
 * @param payload The message.
 */
func (c *Connection) write(mt int, payload []byte) error {
  c.ws.SetWriteDeadline(time.Now().Add(writeWait))
  return c.ws.WriteMessage(mt, payload)
}

/* @brief Write a message if there is one, otherwise ping the client. */
func (c *Connection) writer() {
  ticker := time.NewTicker(pingPeriod)
  defer c.Close()
  for {
    select {
    case m, ok := <-c.send:
      if !ok {
        c.write(websocket.CloseMessage, []byte{})
        return
      }
      if err := c.write(websocket.TextMessage, m); err != nil {
        return
      }
    case <-ticker.C:
      if err := c.write(websocket.PingMessage, []byte{}); err != nil {
        return
      }
    }
  }
}
