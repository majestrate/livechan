package main

import (
  "github.com/dchest/captcha"
  "github.com/gographics/imagick/imagick"
  "github.com/gorilla/mux"
  "log"
  "net/http"
)

func main() {

  // make livechan daemon
  daemon := makeDaemon()

  r := mux.NewRouter()
  
  // set up daemon handlers
  r.Path("/options").HandlerFunc(daemon.OptionsServer)
  r.Path("/channels").HandlerFunc(daemon.ChannelServer)
  r.Path("/convos/{convo}").HandlerFunc(daemon.ConvoServer)
  r.Path("/ws/{channel}").HandlerFunc(daemon.WsServer)
  
  r.Path("/{f}").HandlerFunc(daemon.HtmlServer)
  r.Path("/static/{f}").HandlerFunc(daemon.StaticServer)
  r.Path("/captcha.json").HandlerFunc(daemon.CaptchaServer)
  r.Path("/captcha/{f}").Handler(captcha.Server(captcha.StdWidth, captcha.StdHeight))

  // initialize imagick for thumbnails
  log.Println("initialize imagick")
  imagick.Initialize()
  defer imagick.Terminate()
  
  // start server
  log.Println("livechan going up")
  err := http.ListenAndServe(daemon.GetConfig("bind"), r)
  if err != nil {
    log.Fatal("Unable to serve: ", err)
  }
}

