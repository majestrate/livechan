package main

import (
  "crypto/rand"
  "encoding/base64"
  "crypto/sha256"
  "io"
  "log"
  "strings"
  "fmt"
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



// return random bytes
func randbytes(num int) []byte {
  b := make([]byte, num)
  io.ReadFull(rand.Reader, b)
  return b
}

// generate fatal error message
func ErrorMessage(err error) string {
  return fmt.Sprintf("The backend threw an error, contact the admin. Please supply the channel name and if possible the ip address used to access the site. Error info: %s", err)
}
