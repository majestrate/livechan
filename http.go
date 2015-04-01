package main

import (
  "github.com/dchest/captcha"
  "github.com/gorilla/sessions"
  "github.com/gorilla/websocket"
  "net/http"
  "strings"
  "log"
  "fmt"
)

var upgrader = websocket.Upgrader{
  ReadBufferSize: 1024,
  WriteBufferSize: 1024,
  CheckOrigin: func(r *http.Request) bool { return true }, // TODO: fix
}


// create session store
var session = sessions.NewCookieStore(randbytes(32))


// check for a session, create one if it does not exist
func obtainSession(w http.ResponseWriter, req *http.Request) *sessions.Session {
  sess, _ := session.Get(req, "livechan")
  // newly made session
  if sess.IsNew {
    sess.Values["user"] = new(User)
  }
  return sess
}

// obtain the session's User Object, create session and user if it does not exist
func obtainSessionUser(w http.ResponseWriter, req *http.Request) *User {
  // obtain session
  sess := obtainSession(w, req)
  // get user
  sess_user, _ := sess.Values["user"]
  // type switch
  switch user := sess_user.(type) {
  case *User:
    // return
    return user
    // fall through probably never happens
  }
  // this probably never happens
  return nil
}

func wsServer(w http.ResponseWriter, req *http.Request) {
  sess_user := obtainSessionUser(w, req)
  channelName := req.URL.Path[4:] // Slice off "/ws/"
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  if (storage.getChatChannelId(channelName) == 0) {
    http.Error(w, "Method not allowed", 405)
    return
  }
  ws, err := upgrader.Upgrade(w, req, nil)
  if err != nil {
    fmt.Println(err)
    return
  }
  c := &Connection{
    send: make(chan []byte, 256),
    ws: ws,
    channelName: channelName,
    ipAddr: req.RemoteAddr,
    user: sess_user,
    }
  h.register <- c

  /* Start a reader/writer pair for the new connection. */
  go c.writer()
  /* Nature of go treats this handler as a goroutine.
     Small optimization to not spawn a new one. */
  c.reader()
  
  // when we end we want to decrement the channel count and deregister
  h.unregister <- c
}

func channelServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  fmt.Fprintf(w, "%+v", storage.getChannels());
}

func convoServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  fmt.Fprintf(w, "%+v %s", storage.getConvos(req.URL.Path[8:]), req.URL.Path[8:]);
}

func handleRegistrationPage(w http.ResponseWriter, req *http.Request) {
  http.Error(w, "No registration pages, yet!", 404)
}

func htmlServer(w http.ResponseWriter, req *http.Request) {
  channelName := req.URL.Path[1:] // Omit the leading "/"

  /* Disallow / in the name. */
  if strings.Contains(channelName, "/") {
    http.Error(w, "Method not allowed", 405)
    return
  }

  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }

  if channelName == "" {
    channelName = "General"
  }

  if (storage.getChatChannelId(channelName) == 0) {
    handleRegistrationPage(w, req)
    return
  }
  _ = obtainSession(w, req)
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  http.ServeFile(w, req, "index.html")
}

func captchaServer(w http.ResponseWriter, req *http.Request) {
  if req.Method == "GET" {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprintf(w, "{captcha: %s}", captcha.New());
    return
  } else if req.Method == "POST" {
    captchaId := req.FormValue("captchaId")
    captchaSolution := req.FormValue("captchaSolution")
    if captcha.VerifyString(captchaId, captchaSolution) {
      log.Println("verified captcha for", req.RemoteAddr)
      // this user has solved the required number of captchas
      sess_user := obtainSessionUser(w, req)
      sess_user.SolvedCaptcha = true
    } else {
      log.Println("failed capcha for", req.RemoteAddr)
    }
  }
}

func staticServer(w http.ResponseWriter, req *http.Request) {
  path := req.URL.Path[1:]
  // prevent file tranversal
  if strings.Contains(path, "..") {
    http.Error(w, "Not Found", 404)
  } else {
    http.ServeFile(w, req, path)
  }
}
