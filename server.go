package main

import (
  "net/http"
  "github.com/dchest/captcha"
  "github.com/gographics/imagick/imagick"
  "log"
  //_ "net/http/pprof"
)

func main() {

  cfg.Validate()
  
  db_type := cfg["db_type"]
  db_url := cfg["db_url"]
  
  // make database
  db  := initDB(db_type, db_url)
  // set storage
  storage = &Database{db:db}

  // ensure tor exits are banned
  if cfg.BanTor() {
    BanTor()
  }

  

  
  // TODO: kinda pointless
  // creds := cfg["admin_creds"]
  // storage.EnsureAdminCreds(creds)

  // run hub
  // TODO: shouldn't hub be made in this method?
  go h.run()
  
  // set up http server handlers
  http.HandleFunc("/options", optionsServer)
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
  err := http.ListenAndServe(cfg["bind"], nil)
  if err != nil {
    log.Fatal("Unable to serve: ", err)
  }
}

