package main

import (
  "encoding/json"
  "path/filepath"
  "time"
  "io"
  "strings"
  "os"
  "log"
  "bytes"
)

// incoming chat request
type InChat struct {
  Convo string
  Name string
  Message string
  File string
  FileName string
}

/* To be stored in the DB. */
type Chat struct {
  IpAddr string
  Name string
  Trip string
  Country string
  Message string
  Count uint64
  Date time.Time
  FilePath string
  FileName string
  FilePreview string
  FileSize string
  FileDimensions string
  Convo string
  UserID string
}

/* To be visible to users. */
type OutChat struct {
  // used to indicate a change in channel size
  UserCount int
  // poster's name
  Name string
  // poster's tripcode
  Trip string
  // country flag
  Country string
  // what was chatted
  Message string
  // (?)
  Count uint64
  // date created
  Date time.Time
  // orignal file path
  FilePath string
  // filename
  FileName string
  // file thumbnail path
  FilePreview string
  // file size string
  FileSize string
  // file dimensions string
  FileDimensions string
  // conversation (thread) in channel
  Convo string
  // for stuff like (you) and (mod)
  Capcode string
  // error messages i.e. mod login / captcha failure / bans
  Error string
}

// parse incoming data into Chat
// send chat down channel
func createChat(data []byte, conn *Connection, chnl chan *Chat) {
  inchat := new(InChat)
  // un marshal json
  var inbuf bytes.Buffer
  inbuf.Write(data)
  dec := json.NewDecoder(&inbuf)
  err := dec.Decode(inchat)
  // zero out decoder and input
  dec = nil
  data = nil
  inbuf.Reset()
  if err != nil {
    inchat = nil
    log.Println(conn.ipAddr, "error creating chat: ", err)
    return
  }
  
  if inchat.Empty() {
    inchat = nil
    log.Println("empty post, dropping")
    return
  }
  
  
  c := new(Chat)
  
  // trim name and set to anonymous if unspecified
  c.Name = strings.TrimSpace(inchat.Name)
  if len(c.Name) == 0 {
    c.Name = "Anonymous"
  }

  // trim converstaion (thread) and set to general if unspecified
  c.Convo = strings.TrimSpace(inchat.Convo)
  if len(c.Convo) == 0 {
    c.Convo = "General"
  }
  
  // trim message and set
  c.Message = strings.TrimSpace(inchat.Message)
  // message was recieved now
  c.Date = time.Now().UTC()
  // extract IP address
  // TODO: assumes IPv4
  c.IpAddr = ExtractIpv4(conn.ipAddr)

  // if there is a file present handle upload
  if len(inchat.File) > 0 && len(inchat.FileName) > 0 {
    // TODO FilePreview, FileDimensions
    c.FilePath = genUploadFilename(inchat.FileName)
    c.FileName = inchat.FileName
    log.Println(conn.ipAddr, "uploaded file", c.FilePath)
    handleUpload(inchat, c.FilePath)
  }
  // gc
  inchat = nil
  
  // send it
  chnl <- c
}


// delete files associated with this chat
func (chat *Chat) DeleteFile() {
  log.Println("Delete Chat Files", chat.FilePath)
  os.Remove(filepath.Join("upload", chat.FilePath))
  os.Remove(filepath.Join("thumbs", chat.FilePath))
}

// create json object as bytes
// write to writer
func (chat *OutChat) createJSON(w io.Writer) {
  enc := json.NewEncoder(w)
  err := enc.Encode(chat)
  if err != nil {
    log.Println("error creating json: ", err)
  }
  enc = nil
}

// turn into outchat 
func (chat *Chat) toOutChat() OutChat{
  return OutChat{
    Name: chat.Name,
    Message: chat.Message,
    Date: chat.Date,
    Count: chat.Count,
    Convo: chat.Convo,
    FilePath: chat.FilePath,
  }
}

func (self *InChat) Empty() bool {
  return len(self.Message) == 0 && len(self.File) == 0
}

// create a json array of outchats for an array of chats for a given connection
// write result to a writer
func createJSONs(chats []Chat, w io.Writer) {
  var outChats []OutChat
  for _, chat := range chats {
    outChat := OutChat{
      Name: chat.Name,
      Message: chat.Message,
      Date: chat.Date,
      Count: chat.Count,
      Convo: chat.Convo,
      FilePath: chat.FilePath,
      //Capcode: chat.genCapcode(conn),
    }
    outChats = append(outChats, outChat)
  }
  enc := json.NewEncoder(w)
  err := enc.Encode(outChats)
  if err != nil {
    log.Println("error marshalling json: ", err)
  }
}

