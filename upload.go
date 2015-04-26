package main

import (
  "bytes"
  "io"
  "strings"
  "time"
  "fmt"
  "encoding/base64"
  "path/filepath"
  "log"
)

// generate filename for upload
// TODO: use go stdlib
func genUploadFilename(filename string) string {
  // FIXME invalid filenames without extension
  // get time
  timeNow := time.Now()
  // get extension
  idx := strings.LastIndex(filename, ".")
  // concat time and file extension
  fileExt := filename[idx+1:]
  fname := fmt.Sprintf("%d.%s", timeNow.UnixNano(), fileExt)
  return filepath.Clean(fname)
}

// handle file upload
func handleUpload(chat *InChat, fname string) {

  var inbuff, outbuff bytes.Buffer
  io.WriteString(&inbuff, chat.File)
  dec := base64.NewDecoder(base64.StdEncoding, &inbuff)
  // get the path for the original image
  osfname := filepath.Join("upload", fname)
  // get the path for the thumbnail
  thumbnail := filepath.Join("thumbs", fname)
  _, err := io.Copy(&outbuff, dec)
  dec = nil
  if err != nil {
    log.Println("upload fail in decoding base64", err)
    return
  }
  // clear out input buffer
  inbuff.Reset()
  // get the data
  data := outbuff.Bytes()
  // clear the decoded input buffer
  outbuff.Reset()
  // process image
  err = processImage(fname, osfname, thumbnail, data)
  // clear data buffer
  data = nil
  if err != nil {
    log.Println("failed processing upload", err)
  }
}
