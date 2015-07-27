package main

import (
  "bytes"
  "github.com/gorilla/websocket"
  "log"
  "net"
  "time"
)

const (
  writeWait = 10 * time.Second     // Write timeout.
  pongWait = 60 * time.Second      // Read timeout.
  pingPeriod = (pongWait * 9) / 10 // How frequently to ping the clients.
  maxMessageSize = 1024 * 1024     // Maximum size of a message.
)


// Connection is used to control 1 websocket connection
type Connection interface {
  // our user
  User() *User
  // chan others use to send a chat to us
  Chan() chan OutChat
  //return true if we require this connection to obey cooldown rules
  RequireCooldown() bool
  // get our ip address
  Addr() net.Addr
  // close this connection
  Close()
  // run mainloop
  Run()
  // mark this connection as solved the captcha
  MarkSolvedCaptcha()
  // see if we solved the captcha
  SolvedCaptcha() bool
}

// implemenation of Connection
type liveConnection struct {
  // underlying websocket
  ws *websocket.Conn
  // for chat processing
  chatio ChatIO
  // our user
  user *User
  // chan for sending out chat to websocket
  sendChnl chan OutChat
  // chan for recving in chat from websocket
  inchatChnl chan InChat
  // the chan for the channel we are subscribed to
  chnlChnl chan ChannelChat
}

func (self liveConnection) SolvedCaptcha() bool {
  return self.user.SolvedCaptcha
}

func (self liveConnection) MarkSolvedCaptcha() {
  self.user.MarkSolvedCaptcha()
}

func (self liveConnection) User() *User {
  return self.user
}

func (self liveConnection) RequireCooldown() bool {
  return self.user.RequireCooldown()
}

func (self liveConnection) Chan() chan OutChat {
  return self.sendChnl
}

func (self liveConnection) Close() {
  self.ws.Close()
  close(self.sendChnl)
  close(self.inchatChnl)
}

// run connection mainloop
func (self liveConnection) Run() {
  // set up websocket parameters
  self.ws.SetReadLimit(maxMessageSize)
  self.ws.SetReadDeadline(time.Now().Add(pongWait))
  self.ws.SetPongHandler(func(string) error {
    self.ws.SetReadDeadline(time.Now().Add(pongWait))
    return nil
  })

  go self.poll()
  // new time ticker for pings
  ticker := time.NewTicker(pingPeriod)
  
  for {
    select {
    case <-ticker.C:
      if err := self.write(websocket.PingMessage, []byte{}); err != nil {
        log.Println("failed ping", self.Addr())
        return
      }

    case outchat, ok := <- self.sendChnl:
      // we got an outchat
      // turn it into bytes
      if ok {
        var buff bytes.Buffer
        err := self.chatio.WriteChat(outchat, &buff)
        if err == nil {
          // send it
          if err = self.write(websocket.TextMessage, buff.Bytes()); err != nil {
            log.Println("did not write to websocket", err)
            return
          }
        } else {
          log.Println("cannot serialize to outchat", err)
        }
      } else {
        // not okay channel closed?
        return
      }
    case inchat, ok := <- self.inchatChnl:
      if ok {
        // send it to the channel
        ch := self.chatio.ToChat(inchat)
        self.chnlChnl <- ChannelChat{ch, self}
      } else {
        // not okay channel closed?
        return
      }
    }
  }
}

func (self liveConnection) Addr() net.Addr {
  return self.ws.RemoteAddr()
}

// poll for messages
func (self liveConnection) poll() {
  for {
    // read a message
    _, r, err := self.ws.NextReader()
    if err != nil {
      log.Println("websocket read error", err)
      self.Close()
      return
    }
    inchat, err := self.chatio.ReadChat(r)
    if err == nil {
      self.inchatChnl <- inchat
    } else {
      log.Println("invalid inchat from", self.Addr())
    }
  }
}

/* @brief Sends data to the connection.
 *
 * @param mt The type of message.
 * @param payload The message.
 */
func (self liveConnection) write(mt int, payload []byte) error {
  self.ws.SetWriteDeadline(time.Now().Add(writeWait))
  return self.ws.WriteMessage(mt, payload)
}
