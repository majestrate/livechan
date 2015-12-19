package main

import (
  "net/http"
  "github.com/gorilla/mux"
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

  // run hub
  // TODO: shouldn't hub be made in this method?
  go h.run()

  r := mux.NewRouter()
  
  // set up http server handlers
  r.HandleFunc("/captcha.json", captchaServer)
  r.HandleFunc("/options", optionsServer)
  r.HandleFunc("/channels", channelServer)
  r.HandleFunc("/convos/{f}", convoServer)
  r.HandleFunc("/ws/{f}", wsServer)
  r.HandleFunc("/static/theme/{f}", staticServer)
  r.HandleFunc("/static/contrib/{f}", staticServer)
  r.HandleFunc("/static/{f}", staticServer)
  r.HandleFunc("/{f}", htmlServer)
  r.Handle("/captcha/{f}", captcha.Server(captcha.StdWidth, captcha.StdHeight))
  r.HandleFunc("/{f}", htmlServer)
  

  // ensure that initial channels are there
  storage.ensureChannel("General")

  storage.EnsureAdminCreds(cfg["admin_creds"])

  
  // initialize imagick for thumbnails
  log.Println("initialize imagick")
  imagick.Initialize()
  defer imagick.Terminate()
  
  // start server
  log.Println("livechan going up")
  err := http.ListenAndServe(cfg["bind"], r)
  if err != nil {
    log.Fatal("Unable to serve: ", err)
  }
}

