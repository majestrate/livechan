package livechan

import (
  "bytes"
  "github.com/gorilla/websocket"
  "log"
  "io"
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
  log.Println(c.ipAddr, "close connection")
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
  var r io.Reader
  var err error
  for {
    _, r, err = c.ws.NextReader()
    if err != nil {
      break
    }
    addr := c.user.IpAddr
    // check for global ban
    if storage.isGlobalBanned(addr) {
      // tell them they are banned
      var chat OutChat
      chat.Notify = "Your address (" + addr + ") has been banned globally from Livechan: "
      chat.Notify += storage.getGlobalBanReason(addr)
        // send them the ban notice
      var buff bytes.Buffer
      chat.createJSON(&buff)
      c.send <- buff.Bytes()
    } else if c.user.SolvedCaptcha {
      // copy data into local buffer
      var buff bytes.Buffer
      io.CopyBuffer(&buff, r, nil)
      createChat(buff.Bytes(), c)
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
