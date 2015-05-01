package main

import (
  "encoding/json"
  "encoding/base64"
  "path/filepath"
  "time"
  "io"
  "strings"
  "os"
  "log"
  "bytes"
  "fmt"
)

// incoming chat request
type InChat struct {
  Convo string
  Name string
  Message string
  File string
  FileName string
  ModLogin string // user:password
  ModScope int // moderation request scope
  ModAction int // moderation request action
  ModPostID int // moderation request target
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
  // error messages / mod events / captcha failure / bans
  // pops up a desktop notification
  Notify string
  // the name of an event that has been triggered
  Event string
}

// parse incoming data into Chat
// send chat down channel
func createChat(data []byte, conn *Connection) {
  var inchat InChat
  var buff bytes.Buffer
  // un marshal json
  err := json.Unmarshal(data, &inchat)
  if err != nil {
    log.Println("error creating chat: ", err)
    return 
  }
  
  if inchat.Empty() {
    log.Println("empty post, dropping")
    return 
  }

  var c Chat
  var oc OutChat

  // trim message and set
  c.Message = strings.TrimSpace(inchat.Message)
  
  // attempt a mod login
  if len(inchat.ModLogin) > 0 {
    var username, password string
    idx := strings.Index(inchat.ModLogin, ":")
    username = inchat.ModLogin[:idx]
    password = inchat.ModLogin[1+idx:]

    if conn.user.Login(username, password) {
      oc.Notify = "You have logged in as "+username
      oc.Event = "login:mod";
      // if we are admin, set the event to admin login
      if conn.user.IsAdmin() {
        oc.Event = "login:admin";
      }
    } else {
      oc.Notify = "Login Failed"
      oc.Event = "login:fail"
    }
  }
  
  if inchat.ModScope > 0 && inchat.ModAction > 0 {
    // attempt mod action
    res := conn.user.Moderate(inchat.ModScope, inchat.ModAction, conn.channelName, inchat.ModPostID, 0)
    if res {
      oc.Notify = "Moderation done"
      c.Trip = "!Mod"
      if conn.user.IsAdmin() {
        c.Trip = "!Admin"
      }
      c.Message = fmt.Sprintf("%s %s >>%d", ScopeString(inchat.ModScope), ActionString(inchat.ModAction), inchat.ModPostID)
    } else {
      oc.Notify = "Invalid Permissions"
    }
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
  
  // message was recieved now
  c.Date = time.Now().UTC()

  c.IpAddr = ExtractIpv4(conn.ipAddr)
  
  // if there is a file present handle upload
  if len(inchat.File) > 0 && len(inchat.FileName) > 0 {
    // TODO FilePreview, FileDimensions
    c.FilePath = genUploadFilename(inchat.FileName)
    c.FileName = inchat.FileName
    // decode base64
    dec := make([]byte, base64.StdEncoding.DecodedLen(len(inchat.File)))
    base64.StdEncoding.Decode(dec, []byte(inchat.File))
    if err == nil {
      log.Println(c.IpAddr, "uploaded file", c.FilePath)
      filepath := c.FilePath
      handleUpload(filepath, dec)
    } else {
      log.Println("failed to decode upload", err)
      oc.Notify = "failed to decode upload"
      dec = nil
      return
    }
    
    dec = nil
  }


  // send any immediate notifications
  if len(oc.Notify) > 0 {
    oc.createJSON(&buff)
    conn.send <- buff.Bytes()
  }
  if len(inchat.ModLogin) == 0 && len(inchat.Message) > 0 {
    // send the chat if it wasn't a mod login
    h.broadcast <- Message{chat: c, conn: conn}
  }
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
    Trip: chat.Trip,
  }
}

func (self *InChat) Empty() bool {
  return len(self.Message) == 0 && len(self.File) == 0 && len(self.ModLogin) == 0 && self.ModPostID == 0
}

// create a json array of outchats for an array of chats for a given connection
// write result to a writer
func createJSONs(chats []Chat, out chan []byte) {
  var outChats []OutChat
  for _, chat := range chats {
    outChat := chat.toOutChat()
    outChats = append(outChats, outChat)
  }
  data, err := json.Marshal(&outChats)
  if err != nil {
    log.Println("error marshalling json: ", err)
  }
  out <- data
}

