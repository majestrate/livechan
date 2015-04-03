package main

import (
  "net/http"
  "github.com/dchest/captcha"
  "github.com/gographics/imagick/imagick"
  "log"
)

func main() {
  // make database
  db  := initDB()
  // set storage
  storage = &Database{db:db}
  
  // ensure tor exits are banned
  //BanTor()

  // run hub
  // TODO: shouldn't hub be made in this method?
  go h.run()
  
  // set up http server handlers
  http.HandleFunc("/channels", channelServer)
  http.HandleFunc("/convos/", convoServer)
  http.HandleFunc("/", htmlServer)
  http.HandleFunc("/ws/", wsServer)
  http.HandleFunc("/static/", staticServer)
  http.HandleFunc("/captcha.json", captchaServer)
  http.Handle("/captcha/", captcha.Server(captcha.StdWidth, captcha.StdHeight))
  
  // initialize imagick for thumbnails
  log.Println("initialize imagick")
  imagick.Initialize()
  defer imagick.Terminate()
  
  // start server
  log.Println("livechan going up")
  err := http.ListenAndServe(":18080", nil)
  if err != nil {
    log.Fatal("Unable to serve: ", err)
  }
}

