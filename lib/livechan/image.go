package livechan

import (
  "errors"
  "github.com/gographics/imagick/imagick"
  "log"
  "io/ioutil"
  "strings"
)

// determine if this file is an image
func fileIsImage(fname string) bool {
  // TODO: use for loop
  fname = strings.ToLower(fname)
  if strings.HasSuffix(fname, ".png") {
    return true
  }
  if strings.HasSuffix(fname, ".jpg") {
    return true
  }
  if strings.HasSuffix(fname, ".jpeg") {
    return true
  }
  if strings.HasSuffix(fname, ".webp") {
    return true
  }
  if strings.HasSuffix(fname, ".gif") {
    return true
  }
  return false
}

// process image from upload
func processImage(infname, outfname, thumbfname string, data []byte) error {
  
  err := generateThumbnail(infname, thumbfname, data)
  if err != nil {
    log.Println("failed to generate thumbnail and write file", err)
    return err
  }
  
  // write out original file in background
  go ioutil.WriteFile(outfname, data, 0644)
  return nil
}

// generate thumbanail
func generateThumbnail(inFname, outFname string, data []byte ) error {
  // TODO: accept video, pdf, audio etc
  // is this an image?
  if fileIsImage(inFname) {
    // yes, generate thumbanil
    return generateImageThumbnail(inFname, outFname, data)
  } else {
    // not acceptable type, return error
    return errors.New("file "+inFname+" is not an image")
  }
}

// generate thumbnail for image
func generateImageThumbnail(inFname, outFname string, data []byte) error {
  // TODO: special case for aGIF
  log.Println("generating thumbnail for", inFname)

  var err error
  // initialize new thumbnailer
  wand := imagick.NewMagickWand()
  defer wand.Destroy()

  // read the image
  err = wand.ReadImageBlob(data)
  if err != nil {
    return err
  }
  // get image dimensions
  w := wand.GetImageWidth()
  h := wand.GetImageHeight()
  
  var thumb_w, thumb_h, scale float64
  
  // calculate scale parameters
  scale = 180
  modifer := scale / float64(w)
  
  thumb_w = modifer * float64(w)
  thumb_h = modifer * float64(h)
  
  // scale it muthafuka
  err = wand.ScaleImage(uint(thumb_w), uint(thumb_h))
  if err != nil {
    log.Println("could not scale image to make thumbnail", err)
    return err
  }
  // write out the thumbnail
  err = wand.WriteImage(outFname)
  return err
}
