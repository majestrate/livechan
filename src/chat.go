package main

import (
  "encoding/json"
  "path/filepath"
  "time"
  "io"
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
  Channel string
  ModLogin string // user:password
  ModScope int // moderation request scope
  ModAction int // moderation request action
  ModPostID int // moderation request target
  ModReason string // for ban reasons
  ModExpire int64 // expiration for bans
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

// chat to be sent to channels
type ChannelChat struct {
  chat Chat
  conn Connection
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

type ChatProcessor interface {
  ToChat(inchat InChat) Chat
  ToOutChat(chat Chat) OutChat
}

type ChatReader interface {
  ReadChat(r io.Reader) (InChat, error)
}

type ChatWriter interface {
  WriteChat(outchat OutChat, wr io.Writer) error
}

type ChatIO interface {
  ChatProcessor
  ChatReader
  ChatWriter
}

// marshall to Chat
/*
func (inchat InChat) ToChat() Chat {
  // attempt a mod login
  if len(inchat.ModLogin) > 0 {
    var username, password string
    idx := strings.Index(inchat.ModLogin, ":")
    if idx > 3 {
      username = inchat.ModLogin[:idx]
      password = inchat.ModLogin[1+idx:]
    }

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
    res := conn.user.Moderate(inchat.ModScope, inchat.ModAction, conn.channelName, inchat.ModPostID, inchat.ModExpire, inchat.ModReason)
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
    return oc
  }
}
*/

// delete files associated with this chat if they exists
func (chat *Chat) DeleteFile() {
  if len(chat.FilePath) > 0 {
    log.Println("Delete Chat Files", chat.FilePath)
    os.Remove(filepath.Join("upload", chat.FilePath))
    os.Remove(filepath.Join("thumbs", chat.FilePath))
  }
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

func (self *Chat) Empty() bool {
  return len(self.Message) == 0 && len(self.FilePath) == 0 
}

// create a json array of outchats for an array of chats for a given connection
// write result to a writer
func createJSONs(chats []Chat, wr io.Writer) {
  var outChats []OutChat
  for _, chat := range chats {
    outChat := chat.toOutChat()
    outChats = append(outChats, outChat)
  }
  data, err := json.Marshal(&outChats)
  if err != nil {
    log.Println("error marshalling json: ", err)
  }
  wr.Write(data)
}

// for sorting chat by date
// implements sort.Interface
type chatDateSorter []Chat

func (self chatDateSorter) Len() int {
  return len(self)
}

func (self chatDateSorter) Less(i, j int) bool {
  return self[i].Date.UnixNano() < self[j].Date.UnixNano()
}

func (self chatDateSorter) Swap(i, j int) {
  ch := self[j]
  self[j] = self[i]
  self[i] = ch
}
