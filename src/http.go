package main

import (
  "github.com/dchest/captcha"
  "github.com/gorilla/sessions"
  "github.com/gorilla/websocket"
  "net/http"
  "strings"
  "log"
  "fmt"
  "io"
  "encoding/json"
  "path/filepath"
)

// websocket upgrader
var upgrader = websocket.Upgrader{
  ReadBufferSize: 1024,
  WriteBufferSize: 1024,
  // check for origin validity
  CheckOrigin: func(r *http.Request) bool { return true }, // TODO: fix
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

// create session store
// seed with random bytes each startup
var session = sessions.NewCookieStore([]byte(cfg["api_secret"]))

func init() {
  session.Options = &sessions.Options{
    Path: cfg["prefix"],
    // 60 minutes sessions
    MaxAge: 1000 * 60,
    // TODO: fix this
    HttpOnly: true,
  }
}


func createUserForSession(sess *sessions.Session, req *http.Request) *User {
  var u User
  u.IpAddr = getRealIP(req)
  sess.Values["user"] = u
  return getUserFromSession(sess, req)
}


func getUserFromSession(sess *sessions.Session, req *http.Request) *User{
  val := sess.Values["user"]
  if u , ok := val.(User); ok {
    return &u;
  }
  // no user
  // create it
  return createUserForSession(sess, req)
}


// check for a session, create one if it does not exist
func obtainSession(w http.ResponseWriter, req *http.Request) *sessions.Session {
  addr := getRealIP(req)
  sess, err := session.Get(req, "livechan")
  if err != nil {
    log.Println(addr, "invalid session", err)
    http.Error(w, "please clear your cookies", 500)
    return nil
  }
  if sess.IsNew {
    sess.ID = NewSalt()
    _ = createUserForSession(sess, req)
    sess.Save(req, w)
    // no cookie?
    if len(req.Header.Get("Cookie")) == 0 {
      // yas no cookie, redirect
      path := cfg["prefix"] + req.URL.Path[1:]
      log.Println(addr, "new session, redirecting to", path)
      http.Redirect(w, req, path, 301)
    } else {
      // nah we got a cookie
      // return with no redirect
      return sess
    }
    return nil
  }
  return sess
}

// websocket server
func wsServer(w http.ResponseWriter, req *http.Request) {
  // only accept GET requests
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  
  // obtain session
  sess := obtainSession(w, req)

  // new session, will redirect
  if sess == nil {
    return
  }
  addr := getRealIP(req)
  path := cfg["prefix"] + req.URL.Path[1:]
  channelName := req.URL.Path[4:] // Slice off "/ws/"
  

  // redirect to channel if ends with /
  if strings.HasSuffix(channelName, "/") {
    http.Redirect(w, req, path[:len(path)-2], 301)
    return
  }
  
  // check for chat existing
  if (storage.getChatChannelId(channelName) == 0) {
    log.Println(addr, "no such channel:", channelName)
    http.Error(w, "Not Found", 404)
    return
  }

  
  u := getUserFromSession(sess, req)

  // everything is gud
  // upgrade to web socket
  ws, err := upgrader.Upgrade(w, req, nil)
  if err != nil {
    log.Println(addr, "failed to upgrade websocket", err)
    return
  }
  
  // make a new connection object having the channel name and user object
  var c Connection
  c = Connection{
    // backlog of 32 messages
    send: make(chan []byte, 32),
    ws: ws,
    channelName: channelName,
    ipAddr: addr,
    user: u,
  }

  // register connection with hub
  h.register <- &c

  go c.reader()
  
  /* Start a reader/writer pair for the new connection. */
  c.writer()
  /* Nature of go treats this handler as a goroutine.
     Small optimization to not spawn a new one. */
  
  // unregister after writer ends
  c.Close()
}

// serve list of channels (?)
func channelServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  w.Header().Set("Content-Type", "text/json; charset=utf-8")
  chans := storage.getChannels()
  fmt.Fprintf(w, "[ ")
  for idx := range(chans) {
    chnl := chans[idx]
    fmt.Fprintf(w, "\"%s\", ", chnl);
  }
  fmt.Fprintf(w, "]")
}

// serve list of converstations in a channel (?)
func convoServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  w.Header().Set("Content-Type", "text/json; charset=utf-8")
  // get the channel nane
  channel := req.URL.Path[8:]
  // get the convos for this channel
  convos := storage.getConvos(channel)

  // encode/write the response
  enc := json.NewEncoder(w)
  enc.Encode(convos)
  
}

// serve registration page
func handleRegistrationPage(w http.ResponseWriter, req *http.Request) {
  // http.Error(w, "This channel is not made and No registration page is made, yet!", 404)
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  f := filepath.Join(cfg["webroot_dir"], "register.html")
  http.ServeFile(w, req, f)
}

// serve root page
func htmlServer(w http.ResponseWriter, req *http.Request) {
  sess := obtainSession(w, req)
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
    prefix := cfg["prefix"]
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
    f := filepath.Join(cfg["webroot_dir"], "index.html")
    http.ServeFile(w, req, f)
    return
  }

  // if the channel does not exist ask for registration
  if (storage.getChatChannelId(channelName) == 0) {
    handleRegistrationPage(w, req)
    return
  }
  
  // write out board.html
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  f := filepath.Join(cfg["webroot_dir"], "board.html")
  http.ServeFile(w, req, f)
}

// serve captcha images and solver
func captchaServer(w http.ResponseWriter, req *http.Request) {
  sess := obtainSession(w, req)
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

    // get remote ip address
    addr :=  getRealIP(req)

    // get pre-existing user from session
    user := getUserFromSession(sess, req)

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
      h.captcha <- user
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
func optionsServer(w http.ResponseWriter, req *http.Request) {
  // do not allow anything but GET method
  if req.Method != "GET" {
    http.Error(w, "Method Not Allowed", 405)
    return
  }
  
  // begin writing json response
  io.WriteString(w, "{ ")

  opts := cfg.Options()
  // for each publicly exposable server option
  for idx := range(opts) {
    // write option out with value
    key := opts[idx]
    // only send option if the config has it
    if cfg.Has(key) {
      fmt.Fprintf(w, " \"%s\" : \"%s\", ", key, cfg[key])
    }
  }

  // terminate dict with empty key/value
  io.WriteString(w, "  \"\" : \"\" ")
  // end json response
  io.WriteString(w, "}")

  
}

// serve static content
func staticServer(w http.ResponseWriter, req *http.Request) {
  // initiate session
  sess := obtainSession(w, req)
  // session redirect
  if sess == nil {
    return
  }
  path := req.URL.Path[7:]
  // prevent file tranversal
  if strings.Contains(path, "..") {
    http.Error(w, "Not Found", 404)
  } else {
    path = filepath.Join(cfg["static_dir"], path)
    http.ServeFile(w, req, path)
  }
}
