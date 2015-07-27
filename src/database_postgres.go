//
// postgres database backend implementation
//
package main


import (
  _ "github.com/lib/pq"
  "bufio"
  "database/sql"
  "io/ioutil"
  "log"
  "net/http"
  "strings"
)

type postgresDatabase struct {
  url string
  conn *sql.DB
}

func (self postgresDatabase) Conn() *sql.DB {
  var err error
  if self.conn == nil {
    self.conn, err = sql.Open("postgres", self.url)
    if err != nil {
      log.Fatalf("cannot establish db connection to postgres backend: %s", err)
    }
  }
  return self.conn
}

func (self postgresDatabase) CreateTables() {

  data , err := ioutil.ReadFile("postgres.sql")
  if err != nil {
    log.Fatalf("cannot read postgres_init.sql: %s", err)
  }
  _, err = self.Conn().Execute(string(data))
  if err != nil {
    log.Fatalf("cannot initialize postgres database: %s", err)
  }
}

func (self postgresDatabase) BanChatUser(chat Chat) error {
  // TODO: implement
  return nil
}

func (self postgresDatabase) DeleteChat(chat Chat) error {
  // TODO: implement
  return nil
}

func (self postgresDatabase) GetChannels() (chnls []string, err error) {
  // TODO: implement
  return nil, nil
}

func (self postgresDatabase) GetConvos(chnlName string) (convos []string, err error) {
  // TODO: implement
  return nil, nil
}

func (self postgresDatabase) GetScrollback(chnl, convo string, limit int) (chats []Chat, err error) {
  return nil, nil
}

func (self postgresDatabase) GetTopChannels(limit int) (chnls []string, err error) {
  return nil, nil
}

// ban all tor exits
func (self postgresDatabase) BanTor() error {
  
  // drop old tor exit list
  log.Println("Drop old Tor Exit List")
  self.Conn().Exec("DELETE FROM GlobalBans WHERE offense = 'Tor Exit'")
  
  // open db transaction
  tx, err := self.Conn().Begin()
  if err != nil {
    log.Printf("Cannot open database transaction: %s", err)
    return err
  }
  
  // obatin list
  log.Println("getting list of all Tor Exits")
  exit_list_url := "https://check.torproject.org/exit-addresses"
  resp, err := http.Get(exit_list_url)
  if err != nil {
	  log.Println("failed to obtain tor exit list", err)
    return err
  }
  defer resp.Body.Close()
  
  // read list
  scanner := bufio.NewScanner(resp.Body)
  for scanner.Scan() {
    line := scanner.Text()
    // extract exit address
    if strings.HasPrefix(line, "ExitAddress") {
      idx := strings.Index(line, " ") 
      line = line[1+idx:]
      idx  = strings.Index(line, " ")
      tor_exit := line[:idx]
      // insert record
      _, err :=  tx.Exec("INSERT INTO GlobalBans(ip, offense, expiration) VALUES (?, ?, ?)", tor_exit, "Tor Exit", -1)
      if err != nil {
        log.Println("failed to insert entry into db", err)
        return err
      }
    }
  }
  // commit
  log.Println("commit to database")
  tx.Commit()
  log.Println("commited")
}
