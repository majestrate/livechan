package main

import (
  "bytes"
  "github.com/gorilla/websocket"
  //"log"
  //"strings"
  "time"
)

const (
  writeWait = 10 * time.Second     // Write timeout.
  pongWait = 60 * time.Second      // Read timeout.
  pingPeriod = (pongWait * 9) / 10 // How frequently to ping the clients.
  maxMessageSize = 1024 * 1024         // Maximum size of a message.
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

func (c *Connection) Close() {
  h.unregister <- c
  c.ws.Close()
}

/* @brief Read until there is an error. */
func (c *Connection) reader() {
  /* Clean up once this function exits (can't read). */
  defer c.Close()
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
    } else {
      //log.Println("got message", mtype);
    }
    if c.user.SolvedCaptcha {
      go createChat(d, c, h.broadcast)
    } else {
      var chat OutChat
      chat.Error = "Please fill in the captcha"
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
    case Message, ok := <-c.send:
      if !ok {
        c.write(websocket.CloseMessage, []byte{})
        return
      }
      if err := c.write(websocket.TextMessage, Message); err != nil {
        return
      }
    case <-ticker.C:
      if err := c.write(websocket.PingMessage, []byte{}); err != nil {
        return
      }
    }
  }
}
