//
// config reader
//
package main

import (
  "github.com/majestrate/configparser"
  "log"
  "strings"
)

var needed_keys = []string{"db_type", "db_url", "ban_tor", "prefix", "bind"}

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
  return []string{"prefix", "scrollback", "cooldown"}
}

func (self LivechanConfig) BanTor() bool {
  return strings.ToUpper(self["ban_tor"]) == "YES"
}

func (self LivechanConfig) Has(key string) bool {
  _, ok := self[key]
  return ok 
}

func (self LivechanConfig) Validate() {
  for k := range(needed_keys) {
    key := needed_keys[k]
    if ! self.Has(key) {
      log.Fatal("don't have config value "+key)
    }
  }
}
