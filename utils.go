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

// check if 2 ip address strings are the same
func Ipv4Same(ipaddr1 string, ipaddr2 string) bool {
  delim := ":"
  return strings.Split(ipaddr1, delim)[0] == strings.Split(ipaddr2, delim)[0]
}

// get the ip address from an address string
func ExtractIpv4(addr string) string {
  return strings.Split(addr, ":")[0]
}

// generate a new salt for hashing
func NewSalt() string {
  data := randbytes(24)
  return base64.URLEncoding.EncodeToString(data)
}

// hash password with salt
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


// ban all tor exits
func BanTor() {
  
  // drop old tor exit list
  log.Println("Drop old Tor Exit List")
  storage.db.Exec("DELETE FROM GlobalBans WHERE offense = 'Tor Exit'")
  
  // open db transaction
  tx, err := storage.db.Begin()
  if err != nil {
    log.Fatal("Cannot open database transaction")
  }
  
  // obatin list
  log.Println("getting list of all Tor Exits")
  exit_list_url := "https://check.torproject.org/exit-addresses"
  resp, err := http.Get(exit_list_url)
  if err != nil {
	  log.Fatal("failed to obtain tor exit list", err)
  }
  defer resp.Body.Close()
  
  // read list
  scanner := bufio.NewScanner(resp.Body)
  for scanner.Scan() {
    line := scanner.Text()
    // extract exit address
    if strings.HasPrefix(line, "ExitAddress") {
      idx := strings.Index(line, " ") 
      line = line[1+idx:]
      idx  = strings.Index(line, " ")
      tor_exit := line[:idx]
      // insert record
      _, err :=  tx.Exec("INSERT INTO GlobalBans(ip, offense, expiration) VALUES (?, ?, ?)", tor_exit, "Tor Exit", -1)
      if err != nil {
        log.Fatal("failed to insert entry into db", err)
      }
    }
  }
  // commit
  log.Println("commit to database")
  tx.Commit()
  log.Println("commited")
}

// return random bytes
func randbytes(num int) []byte {
  b := make([]byte, num)
  io.ReadFull(rand.Reader, b)
  return b
}
