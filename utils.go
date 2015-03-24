package main

import (
  "bufio"
  "crypto/rand"
  "encoding/base64"
  "crypto/sha256"
  "io"
  "log"
  "net/http"
  "strings"
)

func Ipv4Same(ipaddr1 string, ipaddr2 string) bool {
  delim := ":"
  return strings.Split(ipaddr1, delim)[0] == strings.Split(ipaddr2, delim)[0]
}

func ExtractIpv4(addr string) string {
  return strings.Split(addr, ":")[0]
}

func NewSalt() string {
  data := make([]byte, 24)
  io.ReadFull(rand.Reader, data)
  return base64.URLEncoding.EncodeToString(data)
}

func hashPassword(password, salt string) string {
  salt_bytes, err := base64.URLEncoding.DecodeString(salt)
  if err != nil {
    log.Println("failed to unpack salt", salt)
    return ""
  }
  h := sha256.New()
  h.Sum(salt_bytes)
  digest := h.Sum([]byte(password))
  return base64.URLEncoding.EncodeToString(digest)
}

func BanTor() {
  
  log.Println("Drop old Tor Exit List")
  storage.db.Exec("DELETE FROM GlobalBans WHERE offense = 'Tor Exit'")
  
  tx, err := storage.db.Begin()
  if err != nil {
    log.Fatal("Cannot open database transaction")
  }
  
  log.Println("getting list of all Tor Exits")
  exit_list_url := "https://check.torproject.org/exit-addresses"
  resp, err := http.Get(exit_list_url)
  if err != nil {
	  log.Fatal("failed to obtain tor exit list", err)
  }
  defer resp.Body.Close()
  
  scanner := bufio.NewScanner(resp.Body)
  for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "ExitAddress") {
      idx := strings.Index(line, " ") 
      line = line[1+idx:]
      idx  = strings.Index(line, " ")
      tor_exit := line[:idx]
      //log.Println("ban "+tor_exit)
      stmt, err :=  tx.Prepare("INSERT INTO GlobalBans(ip, offense, expiration) VAULES (?, ?, ?)")
      if err != nil {
        log.Fatal("failed to insert entry into db")
      }
      stmt.Exec(tor_exit, "Tor Exit", -1)
    }
  }
  log.Println("commit to database")
  tx.Commit()
  log.Println("commited")
}