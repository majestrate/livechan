package main

import (
  "strings"
  "time"
  "fmt"
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
func handleUpload(fname string, data []byte) {

  // get the path for the original image
  osfname := filepath.Join("upload", fname)
  // get the path for the thumbnail
  thumbnail := filepath.Join("thumbs", fname)
  err := processImage(fname, osfname, thumbnail, data)
  // clear data
  if err != nil {
    log.Println("failed processing upload", err)
  }
}
