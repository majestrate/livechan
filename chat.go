package main

import (
  "encoding/json"
  "path/filepath"
  "time"
  "strings"
  "os"
  "log"
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

// parse incoming data
func createChat(data []byte, conn *Connection) *Chat {
  c := new(Chat)
  inchat := new(InChat)
  // un marshal json
  err := json.Unmarshal(data, inchat)
  if err != nil {
    log.Println(conn.ipAddr, "error creating chat: ", err)
    return nil
  }
  // if there is a file present handle upload
  if len(inchat.File) > 0 && len(inchat.FileName) > 0 {
    // TODO FilePreview, FileDimensions
    c.FilePath = genUploadFilename(inchat.FileName)
    c.FileName = inchat.FileName
    log.Println(conn.ipAddr, "uploaded file", c.FilePath)
    handleUpload(inchat, c.FilePath);
  }
  
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
  c.IpAddr = ExtractIpv4(conn.ipAddr);
  return c
}


// delete files associated with this chat
func (chat *Chat) DeleteFile() {
  os.Remove(filepath.Join("upload", chat.FilePath))
  os.Remove(filepath.Join("thumbs", chat.FilePath))
}

// generate capcode
func (chat *Chat) genCapcode(conn *Connection) string {
  cap := ""
  if ExtractIpv4(conn.ipAddr) == chat.IpAddr {
    cap = "(You)"
  }
  return cap
}

// create json object as bytes
func (chat *OutChat) createJSON() []byte {
  j, err := json.Marshal(chat)
  if err != nil {
    log.Println("error: ", err)
  }
  return j
}

// turn into outchat and create json as bytes
func (chat *Chat) createJSON(conn *Connection) []byte{
  outChat := OutChat{
    Name: chat.Name,
    Message: chat.Message,
    Date: chat.Date,
    Count: chat.Count,
    Convo: chat.Convo,
    FilePath: chat.FilePath,
    Capcode: chat.genCapcode(conn),
  }
  return outChat.createJSON()
}

// create a json array of outchats for an array of chats for a given connection
func createJSONs(chats []Chat, conn * Connection) []byte{
  var outChats []OutChat
  for _, chat := range chats {
    outChat := OutChat{
      Name: chat.Name,
      Message: chat.Message,
      Date: chat.Date,
      Count: chat.Count,
      Convo: chat.Convo,
      FilePath: chat.FilePath,
      Capcode: chat.genCapcode(conn),
    }
    outChats = append(outChats, outChat)
  }
  j, err := json.Marshal(outChats)
  if err != nil {
    log.Println("error marshalling json: ", err)
  }
  return j
}


// check if this connection can broadcast
// TODO: is this the best way?
func (chat *Chat) canBroadcast(conn *Connection) bool{
  // no message? don't broadcast
  if len(chat.Message) == 0 {
    return false
  }
  // time based rate limit
  var t = h.channels[conn.channelName][conn]
  // limit minimum broadcast time to 4 seconds
  // don't broadcast
  if time.Now().Sub(t).Seconds() < 4 {
    return false
  }
  // increment chat count and allow broadcast
  // TODO: move elsewhere?
  h.channels[conn.channelName][conn] = time.Now()
  chat.Count = storage.getCount(conn.channelName) + 1
  return true
}
