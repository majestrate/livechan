package main

import (
  "bytes"
  "io"
  "strings"
  "time"
  "fmt"
  "io/ioutil"
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
  if err != nil {
    log.Println("upload fail in decoding base64", err)
    return
  }
  // generate thumbail
  // write it out
  err = generateThumbnail(fname, thumbnail, outbuff.Bytes())
  if err != nil {
    log.Println("failed to generate thumbnail", err)
    return
  }
  // write out original file
  err = ioutil.WriteFile(osfname, outbuff.Bytes(), 0644)
  if err != nil {
    log.Println("failed to save upload", err);
  }
}
