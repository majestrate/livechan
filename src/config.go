//
// config reader
//
package main

import (
  "github.com/majestrate/configparser"
  "log"
  "strconv"
  "strings"
)

var needed_keys = []string{"db_type", "db_url", "ban_tor", "prefix", "bind", "api_secret", "admin_creds", "upload_dir", "thumbs_dir", "webroot_dir", "static_dir", "convo_limit"}

type LivechanConfig map[string]string

func LoadConfig(fname string) LivechanConfig {
  livechan := make(LivechanConfig)
  config, err := configparser.Read(fname)
  if err != nil {
    log.Fatal(err)
  }
  section, err := config.Section("livechan")
  if err != nil {
    log.Fatal(err)
  }
  for k := range(section.Options()) {
    livechan[k] = section.ValueOf(k)
  }
  return livechan
}

// get publicly exposable server options
func (self LivechanConfig) Options() []string {
  return []string{"prefix", "scrollback", "cooldown", "convo_limit"}
}

func (self LivechanConfig) BanTor() bool {
  return strings.ToUpper(self["ban_tor"]) == "YES"
}

func (self LivechanConfig) Has(key string) bool {
  _, ok := self[key]
  return ok 
}



func (self LivechanConfig) GetInt(key string, fallback int64) int64 {
  val, ok := self[key]
  if ok {
    intval, err := strconv.ParseInt(val, 10, 64)
    if err == nil {
      return intval
    }
  }
  return fallback
}

func (self LivechanConfig) Get(key, fallback string) string{
  val, ok := self[key]
  if ok {
    return val
  }
  return fallback
}

func (self LivechanConfig) Validate() {
  for k := range(needed_keys) {
    key := needed_keys[k]
    if ! self.Has(key) {
      log.Fatal("don't have config value "+key)
    }
  }
}


type ChannelConfig map[string]string


func LoadChannelConfig(fname string) ChannelConfig {
  cfg := make(ChannelConfig)
  config, err := configparser.Read(fname)
  if err != nil {
    log.Fatal(err)
  }
  section, err := config.Section("channel")
  if err != nil {
    log.Fatal(err)
  }
  for k := range(section.Options()) {
    cfg[k] = section.ValueOf(k)
  }
  return cfg
}


func (self ChannelConfig) Has(key string) bool {
  _, ok := self[key]
  return ok 
}

func (self ChannelConfig) GetInt(key string, fallback int64) int64 {
  val := self[key]
  intval, err := strconv.ParseInt(val, 10, 64)
  if err == nil {
    return intval
  }
  return fallback
}
