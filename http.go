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
    return req.RemoteAddr
  }
}

// create session store
// seed with random bytes each startup
var session = sessions.NewCookieStore(randbytes(32))

// check for a session, create one if it does not exist
func obtainSession(w http.ResponseWriter, req *http.Request) *sessions.Session {
  sess, _ := session.Get(req, "livechan")
  if sess.IsNew {
    sess.ID = NewSalt()
    sess.Values["captcha"] = false
    sess.Save(req, w)
  }
  return sess
}

// check if this session has already solved a captcha
func sessionSolvedCaptcha(w http.ResponseWriter, req *http.Request) bool {
  sess := obtainSession(w, req)
  capt := sess.Values["captcha"]
  switch val := capt.(type) {
  case bool:
    return val
  default:
    // this should never happen
    return false
  }
}

// websocket server
func wsServer(w http.ResponseWriter, req *http.Request) {
  channelName := req.URL.Path[4:] // Slice off "/ws/"
  // only accept GET requests
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }

  // redirect to channel if ends with /
  if strings.HasSuffix(channelName, "/") {
    http.Redirect(w, req, req.URL.Path[:len(req.URL.Path)-2], 301)
    return
  }
  
  // check for chat existing
  if (storage.getChatChannelId(channelName) == 0) {
    http.Error(w, "Not Found", 404)
    return
  }

  // obtain session
  sess := obtainSession(w, req)
  if sess.IsNew {
    // if this connection had no existing session attached to it something is wrong
    // deny further action
    log.Println("failed to make new websocket, invalid session state")
    http.Error(w, "Method not allowed", 405)
    return
  }

  // upgrade to web socket
  ws, err := upgrader.Upgrade(w, req, nil)
  if err != nil {
    fmt.Println(err)
    return
  }
  // make a new user
  user := new(User)
  // check if this user has already solved a captcha
  if sessionSolvedCaptcha(w, req) {
    // mark them as solved already
    user.MarkSolvedCaptcha()
  }
  // mark this user as having the session's SessionID
  user.Session = sess.ID
  // make a new connection object having the channel name and user object
  var c = Connection{
    send: make(chan []byte, 256),
    ws: ws,
    channelName: channelName,
    ipAddr: getRealIP(req),
    user: user,
  }

  // register connection with hub
  h.register <- &c

  go c.reader()
  
  /* Start a reader/writer pair for the new connection. */
  c.writer()
  /* Nature of go treats this handler as a goroutine.
     Small optimization to not spawn a new one. */
  
  // unregister after reader ends
  h.unregister <- &c
}

// serve list of channels (?)
func channelServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  fmt.Fprintf(w, "%+v", storage.getChannels());
}

// serve list of converstations in a channel (?)
func convoServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  fmt.Fprintf(w, "%+v %s", storage.getConvos(req.URL.Path[8:]), req.URL.Path[8:]);
}

// serve registration page
func handleRegistrationPage(w http.ResponseWriter, req *http.Request) {
  http.Error(w, "This channel is not made and No registration page is made, yet!", 404)
}

// serve root page
func htmlServer(w http.ResponseWriter, req *http.Request) {
  _ = obtainSession(w, req)
  channelName := req.URL.Path[1:] // Omit the leading "/"
  
  /* Disallow / in the name. */
  if strings.Contains(channelName, "/") {
    // redirect to channel
    idx :=  strings.Index(channelName, "/")
    channelName = channelName[:idx]
    prefix := cfg["prefix"]
    http.Redirect(w, req, prefix+channelName, 302)
    return
  }

  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }

  // default to "General" chat if nothing specified
  if channelName == "" {
    channelName = "General"
  }

  // if the channel does not exist ask for registration
  if (storage.getChatChannelId(channelName) == 0) {
    handleRegistrationPage(w, req)
    return
  }
  
  // write out index.html
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  http.ServeFile(w, req, "index.html")
}

// serve captcha images and solver
func captchaServer(w http.ResponseWriter, req *http.Request) {
  if req.Method == "GET" {
    // we are requesting a new captcha
    // TODO: ratelimit
    w.Header().Set("Content-Type", "text/json; charset=utf-8")
    fmt.Fprintf(w, "{\"captcha\": \"%s\"}", captcha.New())
    return
  } else if req.Method == "POST" {
    addr :=  getRealIP(req)
    // we are solving a requested captcha
    responseCode := 0
    captchaId := req.FormValue("captchaId")
    captchaSolution := req.FormValue("captchaSolution")
    if captcha.VerifyString(captchaId, captchaSolution) {

      // this user has solved the captcha
      log.Println("verified captcha for", addr)

      // obtain our session object
      sess := obtainSession(w, req)
      // tell hub that this guy solved a captcha via this session ID
      h.captcha <- sess.ID
      // we succeeded so set response code to 1
      responseCode = 1
      // set session as solved captcha
      sess.Values["captcha"] = true
      // save session states
      sess.Save(req, w)
    } else {
      // failed captcha
      // don't do shit
      log.Println("failed capcha for", addr)
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
  path := req.URL.Path[1:]
  // prevent file tranversal
  if strings.Contains(path, "..") {
    http.Error(w, "Not Found", 404)
  } else {
    http.ServeFile(w, req, path)
  }
}
