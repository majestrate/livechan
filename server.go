package main

import (
  "github.com/gorilla/websocket"
  "net/http"
  "time"
  "log"
  "fmt"
  "bufio"
  "io"
  "os"
  "strings"
)

const (
  writeWait = 10 * time.Second
  pongWait = 60 * time.Second
  pingPeriod = (pongWait * 9) / 10
  maxMessageSize = 512
)

type Connection struct {
  ws *websocket.Conn
  send chan []byte
  channelName string
  ipAddr string
}

type Message struct {
  data []byte
  conn *Connection
}

type Hub struct {
  channels map[string]map[*Connection]time.Time
  broadcast chan Message
  register chan *Connection
  unregister chan *Connection
}

var h = Hub {
  broadcast: make(chan Message),
  register: make(chan *Connection),
  unregister: make(chan *Connection),
  channels: make(map[string]map[*Connection]time.Time),
}

func (h *Hub) run() {
  for {
    select {
    case c := <-h.register:
      if (h.channels[c.channelName] == nil) {
        h.channels[c.channelName] = make(map[*Connection]time.Time)
      }
      h.channels[c.channelName][c] = time.Unix(0,0)
      c.send <- createJSONs(getChats(c.channelName))
    case c := <-h.unregister:
      if _, ok := h.channels[c.channelName][c]; ok {
        delete(h.channels[c.channelName], c)
        close(c.send)
      }
    case m := <-h.broadcast:
      var chat = createChat(m.data, m.conn);
      fmt.Printf("%+v\n", chat);
      if (canBroadcast(chat, m.conn)) {
        for c := range h.channels[m.conn.channelName] {
          select {
          case c.send <- createJSON(chat):
          default:
            close(c.send)
            delete(h.channels[m.conn.channelName], c)
          }
        }
        insertChat(m.conn.channelName, *chat)
      }
    }
  }
}

var upgrader = websocket.Upgrader{
  ReadBufferSize: 1024,
  WriteBufferSize: 1024,
}

func (c *Connection) readPump() {
  defer func() {
    h.unregister <- c
    c.ws.Close()
  }()
  c.ws.SetReadLimit(maxMessageSize)
  c.ws.SetReadDeadline(time.Now().Add(pongWait))
  c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
  for {
    _, d, err := c.ws.ReadMessage()
    if err != nil {
      break
    }
    m := Message{data:d, conn:c}
    h.broadcast <- m
  }
}

func (c *Connection) write(mt int, payload []byte) error {
  c.ws.SetWriteDeadline(time.Now().Add(writeWait))
  return c.ws.WriteMessage(mt, payload)
}

func (c *Connection) writePump() {
  ticker := time.NewTicker(pingPeriod)
  defer func() {
    ticker.Stop()
    c.ws.Close()
  }()
  for {
    select {
    case Message, ok := <-c.send:
      if !ok {
        c.write(websocket.CloseMessage, []byte{})
        return
      }
      if err := c.write(websocket.TextMessage, Message); err != nil {
        return
      }
    case <-ticker.C:
      if err := c.write(websocket.PingMessage, []byte{}); err != nil {
        return
      }
    }
  }
}

func wsServer(w http.ResponseWriter, req *http.Request) {
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  ws, err := upgrader.Upgrade(w, req, nil)
  if err != nil {
    log.Println(err)
    return
  }
  c := &Connection{send: make(chan []byte, 256), ws: ws}
  c.channelName = req.URL.Path[1:]
  c.ipAddr = req.RemoteAddr
  h.register <- c
  go c.writePump()
  c.readPump()
}

func htmlServer(w http.ResponseWriter, req *http.Request) {
  if req.URL.Path != "/" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  if req.Method != "GET" {
    http.Error(w, "Method not allowed", 405)
    return
  }
  serve_file(w, req, "index.html")
} 

func serve_file(w http.ResponseWriter, req *http.Request, filename string) {
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  http.ServeFile(w, req, filename)
}

func staticServer(w http.ResponseWriter, req *http.Request) {
    http.ServeFile(w, req, req.URL.Path[1:])
}

/**
check for session cookie
*/
func checkForSessionCookie(req * http.Request) bool {
  // assume it is there
  return true
  //FIXME it's not there
}


/**
 *
 * handle file upload
 * rolling my own? why not! what is the worst that will go wrong?!
 */
func handleUpload(w http.ResponseWriter, req * http.Request) {
  // tell snoopers to fux off
  if req.Method == "GET" {
    serve_file(w, req, "postform.html")
    return
  }
  // we got a post request
  if req.Method == "POST" {
    // get file upload
    upfile, upheader, err := req.FormFile("file-upload")
    
    if err != nil {
      // we have no file
      w.WriteHeader(403)
			w.Write([]byte(fmt.Sprintf("Upload Error %s\n", err)))
      return
    }
    // we have ensured we have the file now
    fname := upheader.Filename
    // process upload
    doHandleUpload(fname, upfile, w, req)
  }
}

/*
generate uploaded file filename
*/
func genUploadFilename(filename string) string {
  // FIXME invalid filenames without extension
  // get time
  timeNow := time.Now()
  // get extension
  idx := strings.LastIndex(filename, ".")
  // concat time and file extension
  fileExt := filename[idx+1:]
  return fmt.Sprintf("%d.%s", timeNow.UnixNano(), fileExt)
}

/*
 check if a filename is valid
 now only checks extensions
*/
func checkUploadFilenameValid(filename string) bool {
  return ! strings.Contains(filename, "..") &&
    ! strings.Contains(filename, "/") &&
    strings.HasSuffix(filename, ".jpeg") ||
    strings.HasSuffix(filename, ".jpg") ||
    strings.HasSuffix(filename, ".gif") ||
    strings.HasSuffix(filename, ".png") // add more?
}

/*
handle actual file upload

*/
func doHandleUpload(filename string, f io.Reader, w http.ResponseWriter, req * http.Request) {
  // get filename
	if ! checkUploadFilenameValid(filename) {
		log.Printf("Invalid upload filename: %s\n")
		w.WriteHeader(405)
		w.Write([]byte("invalid filename"))
		return
	}
  outFile := genUploadFilename(filename)
  osFile := fmt.Sprintf("upload/%s", outFile)
  log.Printf("upload to: %s\n", osFile)
  // open reader for reading file
  r := bufio.NewReader(f)
  // open file from disk
  file , err := os.Create(osFile)
  if err != nil {
    log.Printf("cannot open outfile %s: %s\n", outFile, err)
    w.WriteHeader(500)
    return
  }
  // open writer for writing to disk
  outWrite := bufio.NewWriter(file)
  // write to disk
  r.WriteTo(outWrite)
  file.Close()
  // tell the world we're gud
  w.WriteHeader(200)
  // spit out uploaded file name
  w.Write([]byte(outFile))
}


func main() {
  go h.run()
  http.HandleFunc("/", htmlServer)
  http.HandleFunc("/ws/", wsServer)
  http.HandleFunc("/static/", staticServer)
  http.HandleFunc("/post/", handleUpload)
  err := http.ListenAndServe(":8080", nil)
  if err != nil {
    log.Fatal("ListenAndServ: ", err)
  }
}

