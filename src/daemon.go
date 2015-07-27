//
// daemon.go -- main logic daemon
//
package main

import (
  "github.com/dchest/captcha"
  "github.com/gorilla/sessions"
  "github.com/gorilla/websocket"
  "encoding/json"
  "fmt"
  "io"
  "log"
  "net/http"
  "path/filepath"
  "strings"
  "time"
)

// interface for livechan core 
type LivechanCore interface {
  // http handlers
  OptionsServer(wr http.ResponseWriter, r *http.Request)
  ChannelServer(wr http.ResponseWriter, r *http.Request)
  StaticServer(wr http.ResponseWriter, r *http.Request)
  HtmlServer(wr http.ResponseWriter, r *http.Request)
  CaptchaServer(wr http.ResponseWriter, r *http.Request)
  ConvoServer(wr http.ResponseWriter, r *http.Request)
  WsServer(wr http.ResponseWriter, r *http.Request)
  // check origin for websockets
  CheckOrigin(r *http.Request) bool

  // get a config value
  GetConfig(key string) string
  
}

// livechan core implementation
type liveCore struct {
  db Database
  upgrader websocket.Upgrader
  session *sessions.CookieStore
  config LivechanConfig
  defaultChannelConfig ChannelConfig
  chat ChatIO
  channels map[string]Channel
  // active users
  users map[string]*User
}




func makeDaemon() LivechanCore {
  var daemon liveCore
  config := LoadConfig("livechan.ini")
  daemon.config = config
  daemon.defaultChannelConfig = LoadChannelConfig("livechan.ini")
  
  daemon.db = makeDatabase(config["db_type"], config["db_url"])
  daemon.db.CreateTables()
  // websocket upgrader
  daemon.upgrader = websocket.Upgrader{
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
    // check for origin validity
    CheckOrigin: daemon.CheckOrigin,
  }
  // create session store
  daemon.session = sessions.NewCookieStore([]byte(daemon.config["api_secret"]))
  daemon.session.Options = &sessions.Options{
    Path: daemon.config["prefix"],
    // 60 minutes sessions
    MaxAge: 1000 * 60,
    // TODO: fix this
    HttpOnly: true,
  }
  // create channel map
  daemon.channels = make(map[string]Channel)
  return daemon
}

func (self liveCore) GetConfig(key string) string {
  return self.config.Get(key, "")
}

func (self liveCore) CheckOrigin(r *http.Request) bool {
  // TODO: implement
  return true
}

// if we have x-forwarded-for header use that
// otherwise use remote address
// in nginx you need to set this in your reverse proxy settings
//
//  loaction / {
//    proxy_set_header X-Real-IP $remote_addr;
//    proxy_pass http://127.0.0.1:18080/
//  }
//
func getRealIP(req * http.Request) string {
  ip := req.Header.Get("X-Real-IP")
  if len(ip) > 0 {
    return ip
  } else {
    return ExtractIpv4(req.RemoteAddr)
  }
}

// create a new channel given name
// does nothing if it already exists
func (self liveCore) newChannel(name string) Channel {
  if self.hasChannel(name) {
    return nil
  }
  // make it
  chnl := liveChannel{
    connections: make(map[Connection]time.Time),
    name: name,
    chnlChatChnl: make(chan ChannelChat),
    config: self.getChannelConfig(name),
  }
  // put it
  self.channels[name] = chnl
  // run it
  go chnl.Run()
  return chnl
}

func (self liveCore) getChannelConfig(name string) ChannelConfig {
  // TODO: implement
  return self.defaultChannelConfig
}

// return true if the channel exists
// otherwise false
func (self liveCore) hasChannel(name string) bool {
  _, ok := self.channels[name]
  return ok
}

// get the a channel's chan for chat
// make a new channel if it doesn't exist
func (self liveCore) getChannelChan(name string) chan ChannelChat {
  // create if not found
  if ! self.hasChannel(name) {
    _ = self.newChannel(name)
  }
  // get
  chnl, _ := self.channels[name]
  // return chan
  return chnl.Chan()
}


// make a new connection
func (self liveCore) newConnection(ws *websocket.Conn, chnlName string, user *User) Connection {
  // make a new connection object having the channel name and user object
  c := liveConnection{
    // backlog of 32 messages
    sendChnl: make(chan OutChat, 32),
    inchatChnl: make(chan InChat, 8),
    chnlChnl: self.getChannelChan(chnlName),
    ws: ws,
    chatio: self.chat,
    user: user,
  }
  return c
}

func (self liveCore) obtainSession(w http.ResponseWriter, req *http.Request) *sessions.Session {
  return nil //TODO: implement
}

// websocket server
func (self liveCore) WsServer(w http.ResponseWriter, req *http.Request) {
  // only accept GET requests
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }

  // get underlying session
  sess := self.obtainSession(w, req)
  
  // new session, will redirect
  if sess == nil {
    return
  }
  addr := getRealIP(req)
  path := self.config["prefix"] + req.URL.Path[1:]
  channelName := req.URL.Path[4:] // Slice off "/ws/"
  

  // redirect to channel if ends with /
  if strings.HasSuffix(channelName, "/") {
    http.Redirect(w, req, path[:len(path)-2], 301)
    return
  }

  if ! self.hasChannel(channelName) {
    log.Println("no such channel for websocket server", channelName)
    http.Error(w, "Not Found", 404)
    return
  }
  
  // everything is gud
  // upgrade to web socket
  ws, err := self.upgrader.Upgrade(w, req, nil)
  if err != nil {
    log.Println(addr, "failed to upgrade websocket", err)
    return
  }

  // get user info
  user := self.getUserForAddr(addr)

  // make it
  c := self.newConnection(ws, channelName, user)

  // run it
  c.Run()
  
  // close it
  log.Println("close connection")
  c.Close()
  log.Println("connection closed")
}

func (self liveCore) getUserForAddr(addr string) *User {
  if _, ok := self.users[addr] ; ! ok {
    self.users[addr] = CreateUser(addr)
  }
  return self.users[addr]
}

// get all active channels
func (self liveCore) getChannels() []string {
  var chans []string
  for chnl := range(self.channels) {
    chans = append(chans, chnl)
  }
  return chans
}

// serve list of channels (?)
func (self liveCore) ChannelServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  w.Header().Set("Content-Type", "text/json; charset=utf-8")
  chans := self.getChannels()
  // encode/write the response
  enc := json.NewEncoder(w)
  enc.Encode(chans)    
}

// serve list of converstations in a channel (?)
func (self liveCore) ConvoServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  w.Header().Set("Content-Type", "text/json; charset=utf-8")
  // get the channel nane
  channel := req.URL.Path[8:]
  // get the convos for this channel
  convos, err := self.db.GetConvos(channel)
  if err == nil {
    // encode/write the response
    enc := json.NewEncoder(w)
    enc.Encode(convos)
  } else {
    http.Error(w, "Internal Server Error", 500)
    io.WriteString(w, fmt.Sprintf("error: %s", err))
  }
  
}

// serve registration page
func (self liveCore) handleRegistrationPage(w http.ResponseWriter, req *http.Request) {
  // http.Error(w, "This channel is not made and No registration page is made, yet!", 404)
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  f := filepath.Join(self.config["webroot_dir"], "register.html")
  http.ServeFile(w, req, f)
}

// serve root page
func (self liveCore) HtmlServer(w http.ResponseWriter, req *http.Request) {
  sess := self.obtainSession(w, req)
  // check for new session
  if sess == nil {
    return
  }
  channelName := req.URL.Path[1:] // Omit the leading "/"
  
  /* Disallow / in the name. */
  if strings.Contains(channelName, "/") {
    // redirect to channel
    idx :=  strings.Index(channelName, "/")
    channelName = channelName[:idx]
    prefix := self.config["prefix"]
    http.Redirect(w, req, prefix+channelName, 301)
    return
  }

  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }

  // show landing page if no channel
  if channelName == "" {
    // write out index.html
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    f := filepath.Join(self.config["webroot_dir"], "index.html")
    http.ServeFile(w, req, f)
    return
  }

  // if the channel does not exist ask for registration
  if ! self.hasChannel(channelName) {
    self.handleRegistrationPage(w, req)
    return
  }
  
  // write out board.html
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  f := filepath.Join(self.config["webroot_dir"], "board.html")
  http.ServeFile(w, req, f)
}

// serve captcha images and solver
func (self liveCore) CaptchaServer(w http.ResponseWriter, req *http.Request) {
  sess := self.obtainSession(w, req)
  // check for new session
  if sess == nil {
    return
  }
  if req.Method == "GET" {
    // we are requesting a new captcha
    // TODO: ratelimit
    w.Header().Set("Content-Type", "text/json; charset=utf-8")
    fmt.Fprintf(w, "{\"captcha\": \"%s\"}", captcha.New())
    return
  } else if req.Method == "POST" {
    addr := getRealIP(req)
    user := self.getUserForAddr(addr)
    if user.IpAddr != addr  {
      log.Printf("ip mismatch %s!=%s", user.IpAddr, addr)
      // possible spoofing or harvesting or something bad
      // send bogus 418 teapot response to fuck with it :3
      http.Error(w, "Am I Kawaii uguu~?", 418)
      return
    }
    
    // we are solving a requested captcha
    responseCode := 0
    captchaId := req.FormValue("captchaId")
    captchaSolution := req.FormValue("captchaSolution")
    if captcha.VerifyString(captchaId, captchaSolution) {

      // this user has solved the captcha
      log.Println(addr, "verified captcha")
      // we succeeded so set response code to 1
      responseCode = 1
      // inform hub of captcha success
      user.MarkSolvedCaptcha()
    } else {
      // failed captcha
      // don't do shit
      log.Println(addr, "failed capcha")
    }
    // write response
    fmt.Fprintf(w, "{\"solved\" : %d }", responseCode)
  }
}

// get livechan chat options
func (self liveCore) OptionsServer(w http.ResponseWriter, req *http.Request) {
  // do not allow anything but GET method
  if req.Method != "GET" {
    http.Error(w, "Method Not Allowed", 405)
    return
  }
  
  // begin writing json response
  io.WriteString(w, "{ ")

  opts := self.config.Options()
  response_opts := make(map[string]string)
  // for each publicly exposable server option
  for _, key := range(opts) {
    if self.config.Has(key) {
      val := self.config[key]
      response_opts[key] = val
    }
  }

  enc := json.NewEncoder(w)
  enc.Encode(response_opts)
}

// serve static content
func (self liveCore) StaticServer(w http.ResponseWriter, req *http.Request) {
  // obtain session and redirect as needed
  sess := self.obtainSession(w, req)
  if sess == nil {
    return
  }
  path := req.URL.Path[7:]
  // prevent file tranversal
  if strings.Contains(path, "..") {
    http.Error(w, "Not Found", 404)
  } else {
    path = filepath.Join(self.config["static_dir"], path)
    http.ServeFile(w, req, path)
  }
}
