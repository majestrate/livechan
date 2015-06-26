package main

import (
  "database/sql"
  "fmt"
  _ "github.com/mattn/go-sqlite3"
  "time"
  "log"
)

type Database struct {
  db *sql.DB
}

var storage *Database

func (s *Database) deleteChatForIP(ipaddr string) {
  log.Println("delete all chat for ", ipaddr)
  tx, err := s.db.Begin()
  if err != nil {
    log.Println("Error: could not access DB.", err)
    return
  }
  stmt, err := s.db.Prepare("SELECT file_path FROM Chats WHERE ip = ?")
  rows, err := stmt.Query(&ipaddr)
  defer rows.Close()
  for rows.Next() {
    var chat Chat;
    rows.Scan(&chat.FilePath)
    chat.DeleteFile();
  }
  defer stmt.Close()
  stmt, err = tx.Prepare("DELETE FROM Chats WHERE ip = ?")
  defer stmt.Close()
  if err != nil {
    log.Println("Error: could not access DB.", err)
    return
  }
  _, err = stmt.Query(ipaddr)
  tx.Commit()
}


// get the user attributes / permissions for a global moderator
func (s *Database) getModAttributes(username string) map[string]string {
  attrs := make(map[string]string)
  stmt, err := s.db.Prepare("SELECT name, value FROM UserAttributes WHERE user_id = ( SELECT id FROM Users WHERE name = ? )")
  defer stmt.Close()
  rows, err := stmt.Query(&username)
  if err != nil {
    log.Println("error getting mod attributes", err)
    return attrs
  }
  defer rows.Close()
  for rows.Next() {
    var attr_name, attr_val string
    rows.Scan(&attr_name, &attr_val)
    attrs[attr_name] = attr_val
  }
  return attrs
}

// do a mod event
// does not check permissions
func (s *Database) ProcessModEvent(ev ModEvent) {
  // should these chats be deleted from the database?
  delChats := ev.Action >= ACTION_DELETE_POST
  // should these chats have their files removed?
  delChatFiles := ev.Action >= ACTION_DELETE_FILE

  
  
  tx, err := s.db.Begin()
  if err != nil {
    log.Println("failed to prepare transaction for mod action", err)
    return
  }
  // is this a ban?
  if ev.Action >= ACTION_BAN {
    // ban the fucker
    stmt, err := tx.Prepare("SELECT ip FROM Chats WHERE channel IN ( SELECT id FROM Channels WHERE name = ? ) AND count = ?")
    var ip string
    stmt.QueryRow(ev.ChannelName, ev.PostID).Scan(&ip)
    if len(ip) == 0 {
      log.Println("wtf we can't get the poster's ID")
      return
    }
    defer stmt.Close()
    log.Println("Ban", ip)
    stmt, err = tx.Prepare("INSERT INTO GlobalBans(ip, offense, expiration, date) VALUES(?,?,?,?)")
    if err != nil {
      log.Println("failed to ban", err)
      return
    }
    defer stmt.Close()
    if len(ev.Reason) == 0 {
      ev.Reason = "No Reason Given"
    }
    stmt.Exec(ip, ev.Reason, ev.Expire, time.Now())
    tx.Commit()
  }

  tx, err = s.db.Begin()
  
  
  var queryFile, queryDelete string
  if ev.Scope == SCOPE_GLOBAL && ev.Action >= ACTION_DELETE_ALL {
    // all posts in this for this ip
    queryFile = `SELECT file_path FROM Chats WHERE ip IN ( 
                   SELECT ip FROM Chats WHERE 
                     channel IN (
                       SELECT id FROM Channels WHERE name = ? LIMIT 1
                   ) AND count = ?
                 )`
    queryDelete = `DELETE FROM Chats WHERE ip IN (
                     SELECT ip FROM Chats WHERE 
                       channel IN (
                         SELECT id FROM Channels WHERE name = ? LIMIT 1
                     ) AND count = ?
                   )`
  } else if ev.Scope == SCOPE_CHANNEL && ev.Action >= ACTION_DELETE_ALL {
    // all posts in this channel for this ip
    queryFile = `WITH chan(id) AS ( SELECT id FROM Channels WHERE name = ?)
                 SELECT file_path FROM Chats WHERE channel IN chan
                 AND ip IN ( SELECT ip FROM Chats WHERE channel IN chan AND count = ? )`
    queryDelete = `WITH chan(id) AS ( SELECT id FROM Channels WHERE name = ? ) 
                   DELETE FROM Chats WHERE channel IN chan AND ip IN ( 
                   SELECT ip FROM Chats WHERE channel IN chan AND count = ?)`
  } else {
    // this post explicitly
    queryFile = `SELECT file_path FROM Chats WHERE channel IN (
                   SELECT id FROM Channels WHERE name = ? LIMIT 1
                 )
                 AND count = ? LIMIT 1`
    queryDelete = `DELETE FROM Chats WHERE channel IN (
                     SELECT id FROM Channels WHERE name = ? LIMIT 1
                   )
                   AND count = ?`
  }
  // do we want to delete files?
  if delChatFiles {
    stmt, err := tx.Prepare(queryFile)
    if err != nil {
      log.Println("cannot prepare file selection sql query for SCOPE", ev.Scope, "POST", ev.PostID, err)
      return
    }
    rows, err := stmt.Query(ev.ChannelName, ev.PostID)
    if err != nil {
      log.Println("cannot execute file selection sql query for global delete all", err)
      return
    }
    defer rows.Close()
    // collect results from file query
    for rows.Next() {
      var chat Chat
      rows.Scan(&chat.FilePath)
      if len(chat.FilePath) > 0 {
        // delete files first
        chat.DeleteFile()
      }
    }
  }
  // do we want to delete chats?
  if delChats {
    stmt, err := tx.Prepare(queryDelete)
    if err != nil {
      log.Println("cannot prepare chat delete sql query", err)
      return
    }
    defer stmt.Close()
    _, err = stmt.Exec(ev.ChannelName, ev.PostID)

    if err != nil {
      log.Println("cannot execute chat delete sql query", err)
    }
  }
  // commit transaction
  tx.Commit()
  log.Println("mod event", ev.Scope, ev.Action, ev.ChannelName, ev.PostID)
}

// set a mod's attribute to a given value
// return true if the operation succeeded otherwise false
func (s *Database) setModAttribute(username, attribute_name, attribute_val string) bool {
  tx, err := s.db.Begin()
  if err != nil {
    log.Println("failed to set mod", username, "attribute", attribute_name, "to", attribute_val)
    return false
  }
  stmt, err := tx.Prepare(`
    INSERT OR REPLACE INTO UserAttributes 
    VALUES (
      (SELECT id FROM Users WHERE name = ?),
      ?,
      ?
    )
  `)
  if err != nil {
    log.Println("failed to prepare query for setModAttribute()", err)
    return false
  }
  defer stmt.Close()
  _, err = stmt.Exec(username, attribute_name, attribute_val)
  if err != nil {
    log.Println("failed to execute query for setModAttribute()", err)
    tx.Rollback()
    return false
  }
  tx.Commit()
  return true
}

// check if a moderator login is correct
// return true if it is a valid login otherwise false
func (s *Database) checkModLogin(username, password string) bool {
  
  var salt, passwordHash string
  stmt, err := s.db.Prepare("SELECT salt, password FROM Users WHERE name = ?")
  defer stmt.Close()
  if err != nil {
    log.Println("Error cannot access DB.", err)
    return false
  }
  
  stmt.QueryRow(&username).Scan(&salt, &passwordHash)
  if hashPassword(password, salt) == passwordHash {
    return true
  }
  return false
}

func (s *Database) insertChannel(channelName string) {
  tx, err := s.db.Begin()
  if err != nil {
    log.Println("Error: could not access DB.", err)
    return
  }
  stmt, err := tx.Prepare("INSERT INTO Channels(name) VALUES(?)")
  defer stmt.Close()
  if err != nil {
    log.Println("Error: could not access DB.", err);
    return
  }
  _, err = stmt.Exec(channelName)
  tx.Commit()
  log.Println("commited new channel:", channelName)
}

func (s *Database) insertConvo(channelId int, convoName string) {
  tx, err := s.db.Begin()
  if err != nil {
    log.Println("Error: could not access DB.", err)
    return
  }
  stmt, err := tx.Prepare("INSERT INTO Convos(channel, name) VALUES(?, ?)")
  defer stmt.Close()
  if err != nil {
    log.Println("Error: could not access DB.", err);
    return
  }
  _, err = stmt.Exec(channelId, convoName)
  tx.Commit()
  log.Println("new convo for", channelId, convoName)
}

// ensure that a channel exists
// create the channel if it is not there
func (s *Database) ensureChannel(channelName string) {
  stmt, err := s.db.Prepare("SELECT COUNT(*) FROM Channels WHERE name = ?")
  if err != nil {
    log.Println("cannot prepare query for checking if channel is there", err)
    return
  }
  defer stmt.Close()

  var numChans int64
  stmt.QueryRow(channelName).Scan(&numChans)
  
  if numChans == 0 {
    log.Println("Create Channel", channelName)
    tx, err := s.db.Begin()
    if err != nil {
      log.Println("cannot create transaction for ensuring channel", err)
      return
    }
    // create the channel since it's not there
    stmt, err = tx.Prepare("INSERT INTO Channels(name) VALUES(?)")
    if err != nil {
      log.Println("failed to prepare query to insert channel", err)
      return
    }
    defer stmt.Close()
    _, err = stmt.Exec(channelName)
    if err != nil {
      log.Println("failed to execute query to insert channel", err)
      return
    }
    tx.Commit()
  }
}

func (s *Database) insertChat(chnl *Channel, chat *Chat) {
  /* Get the ids. */
  channelId := s.getChatChannelId(chnl.Name)
  // no such channel ?
  if channelId == 0 {
    s.insertChannel(chnl.Name)
    channelId = s.getChatChannelId(chnl.Name)
    if channelId == 0 {
      log.Println("Error creating channel", chnl.Name);
      return
    }
  }
  convoId := s.getChatConvoId(channelId, chat.Convo)
  if convoId == 0 {
    s.insertConvo(channelId, chat.Convo)
    convoId = s.getChatConvoId(channelId, chat.Convo)
  }

  tx, err := s.db.Begin()
  if err != nil {
    log.Println("Error: could not access DB.", err);
    return
  }
  stmt, err := tx.Prepare(`
  INSERT INTO Chats(
    ip, name, trip, country, message, count, chat_date, 
    file_path, file_name, file_preview, file_size, 
    file_dimensions, convo, channel
  )
  VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
  defer stmt.Close()
  if err != nil {
    log.Println("Error: could not prepare insert chat", err);
    return
  }
  _, err = stmt.Exec(chat.IpAddr, chat.Name, chat.Trip, chat.Country, chat.Message, chat.Count, chat.Date.UnixNano(), chat.FilePath, chat.FileName, chat.FilePreview, chat.FileSize, chat.FileDimensions, convoId, channelId)

  // roll over chat

  stmt, err = tx.Prepare(`SELECT file_path FROM Chats 
                          WHERE chat_date NOT IN ( 
                            SELECT chat_date FROM Chats WHERE convo = ? AND channel = ? ORDER BY chat_date DESC LIMIT ? ) 
                          AND channel = ? AND convo = ?`)
  if err != nil {
    log.Println("cannot prepare file_path SELECT query for chat roll over", err)
    return
  }
  defer stmt.Close()
  rows, err := stmt.Query(convoId, channelId, chnl.Scrollback, channelId, convoId)
  defer rows.Close()
  for rows.Next() {
    var delchat Chat
    rows.Scan(&delchat.FilePath)
    if len(delchat.FilePath) > 0 {
      delchat.DeleteFile()
    }
  }
  
  stmt, err = tx.Prepare(`UPDATE Chats SET file_path = '' WHERE chat_date NOT IN ( 
                            SELECT chat_date FROM Chats WHERE convo = ? AND channel = ? ORDER BY chat_date DESC LIMIT ? ) 
                          AND channel = ? AND convo = ?`)
  if err != nil {
    log.Println("cannot prepare query for reset filepath on old posts", err)
    return
  }
  defer stmt.Close()
  _, err = stmt.Exec(convoId, channelId, chnl.Scrollback, convoId, channelId)
  
  tx.Commit()
  if err != nil {
    log.Println("Error: could not insert chat", err);
    return
  }
}

func (s *Database) getChatConvoId(channelId int, convoName string)int {
  stmt, err := s.db.Prepare("SELECT id FROM Convos WHERE name = ? AND channel = ?")
  if err != nil {
    log.Println("Error: could not access DB.", err);
    return 0
  }
  defer stmt.Close()
  var id int
  err = stmt.QueryRow(convoName, channelId).Scan(&id)
  if err != nil {
    log.Println("error getting convo id", err)
    return 0
  }
  return id
}

func (s *Database) getChatChannelId(channelName string)int {
  stmt, err := s.db.Prepare("SELECT id FROM Channels WHERE name = ?")
  if err != nil {
    log.Println("Error: could not access DB.", err);
    return 0
  }
  defer stmt.Close()
  var id int
  err = stmt.QueryRow(channelName).Scan(&id)
  if err != nil {
    log.Println("error getting channel id", err)
    return 0
  }
  return id
}

func (s *Database) getCount(channelName string) uint64{
  stmt, err := s.db.Prepare(`
  SELECT MAX(count)
  FROM chats
  WHERE channel = (
    SELECT id FROM channels WHERE name = ?
  )
  `)
  if err != nil {
    log.Println("Couldn't get count.", err)
    return 0
  }
  var count uint64
  stmt.QueryRow(channelName).Scan(&count)
  defer stmt.Close()
  return count
}

func (s *Database) getChannels() []string{
  var outputChannels []string
  stmt, err := s.db.Prepare(`
  SELECT channels.name, MAX(chats.chat_date)
  FROM channels
    left join chats ON chats.channel = channels.id
  GROUP BY channels.name
  ORDER BY chats.chat_date`)
  if err != nil {
    log.Println("Couldn't get channels.", err)
    return outputChannels
  }
  defer stmt.Close()
  rows, err := stmt.Query()
  if err != nil {
    log.Println("Couldn't get channels.", err)
    return outputChannels
  }
  defer rows.Close()
  for rows.Next() {
    var channelDate int
    var channelName string
    rows.Scan(&channelName, &channelDate)
    outputChannels = append(outputChannels, channelName)
  }
  return outputChannels
}

func (s *Database) getConvos(channelName string) []string{
  var outputConvos []string
  stmt, err := s.db.Prepare(`
  SELECT convos.name, MAX(chats.chat_date)
  FROM convos
    left join chats ON chats.convo = convos.id
  WHERE convos.channel = (
    SELECT id FROM channels WHERE name = ?
  )
  GROUP BY convos.name
  ORDER BY chats.chat_date DESC LIMIT 20`)
  if err != nil {
    log.Println("Couldn't get convos.", err)
    return outputConvos
  }
  defer stmt.Close()
  rows, err := stmt.Query(channelName)
  if err != nil {
    log.Println("Couldn't get convos.", err)
    return outputConvos
  }
  defer rows.Close()
  for rows.Next() {
    var convoDate int
    var convoName string
    rows.Scan(&convoName, &convoDate)
    outputConvos = append(outputConvos, convoName)
  }
  return outputConvos
}

func (s *Database) getChats(channelName string, convoName string, numChats uint64) []Chat {
  var outputChats []Chat
  if len(convoName) > 0 {
    stmt, err := s.db.Prepare(`
    SELECT * FROM
    (SELECT ip, chats.name, trip, country, message, count, chat_date,
        file_path, file_name, file_preview, file_size,
        file_dimensions
    FROM chats
    JOIN (SELECT * FROM channels WHERE channels.name = ?)
      AS filtered_channels ON chats.channel=filtered_channels.id
    JOIN (SELECT * FROM convos WHERE convos.name = ?)
      AS filtered_convos ON chats.convo=filtered_convos.id
    ORDER BY COUNT DESC LIMIT ?) ORDER BY COUNT ASC`)
    if err != nil {
      log.Println("Couldn't get chats.", err)
      return outputChats
    }
    defer stmt.Close()
    rows, err := stmt.Query(channelName, convoName, numChats)
    if err != nil {
      log.Println("Couldn't get chats.", err)
      return outputChats
    }
    defer rows.Close()
    for rows.Next() {
      var chat Chat
      var unixTime int64
      rows.Scan(&chat.IpAddr, &chat.Name, &chat.Trip, &chat.Country,
        &chat.Message, &chat.Count, &unixTime, &chat.FilePath,
        &chat.FileName, &chat.FilePreview, &chat.FileSize, &chat.FileDimensions)
      chat.Date = time.Unix(0, unixTime)
      chat.Convo = convoName
      outputChats = append(outputChats, chat)
    }
  }
  return outputChats
}

func (s *Database) getPermissions(channelName string, userName string) uint64 {
  stmt, err := s.db.Prepare(`
  SELECT permissions FROM Owners
  WHERE user = (SELECT id FROM users WHERE name = ?)
  AND channel = (SELECT id FROM channels WHERE name = ?)
  `)
  if err != nil {
    log.Println("Error: could not access DB.", err);
    return 0
  }
  defer stmt.Close()
  var permissions uint64
  err = stmt.QueryRow(userName, channelName).Scan(&permissions)
  if err != nil {
    return 0
  }
  return 0
}

func (s* Database) isGlobalBanned(ipAddr string) bool {
  stmt, err := s.db.Prepare(`
  SELECT COUNT(*) FROM GlobalBans
  WHERE ip = ?
  `)
  if err != nil {
    log.Println("Error: could not access DB for global ban.", err);
    return false
  }
  defer stmt.Close()
  var isbanned int
  err = stmt.QueryRow(ipAddr).Scan(&isbanned)
  if err != nil {
    log.Println("failed to query database for global ban", err)
    return false
  }
  return isbanned > 0
}

func (self *Database) getGlobalBanReason(ipAddr string) string {
  stmt, err := self.db.Prepare(`
  SELECT offense FROM GlobalBans WHERE ip = ?
  `)
  if err != nil {
    log.Println("error with global ban reason query", err)
    return fmt.Sprintf("error: %s", err)
  }
  var reason string
  defer stmt.Close()
  err = stmt.QueryRow(ipAddr).Scan(&reason)
  if err != nil {
    log.Println("error obtaining global ban record", err)
    return fmt.Sprintf("error: %s", err)
  }
  return reason
}

func createTable(db *sql.DB, name, query string) error {
  log.Println("Create table", name)
  _, err := db.Exec(query)
  if err != nil {
    log.Println("Unable to create Table",name, err);
  }
  return err
}


// ensure that the admin login uses the given password
func (self * Database) EnsureAdminCreds(credentials string) {
  log.Println("updating admin credentials")
  salt := NewSalt()
  hash := hashPassword(credentials, salt)
  log.Println(hash)
  tx, err := self.db.Begin()
  if err != nil {
    log.Fatal(err)
  }
  
  stmt, err := tx.Prepare("INSERT INTO Users(name, password, salt) VALUES(?,?,?)")
  if err != nil {
    log.Fatal(err)
  }
  defer stmt.Close()
  _, err = stmt.Exec("admin", hash, salt)
  if err != nil {
    log.Fatal(err)
  }
  tx.Commit()
  log.Println("okay")
  user := CreateUser()
  user.Login("admin", credentials)
  user.GrantAdmin()
  user.Store()
}

func initDB(driver, url string) *sql.DB{
  log.Println("initialize database")
  db, err := sql.Open(driver, url);
  if err != nil {
    log.Println("Unable to open db.", err)
    return nil
  }

  tables := make(map[string]string)
  
  tables["Channels"] = `CREATE TABLE IF NOT EXISTS Channels(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    api_key VARCHAR(255),
    options TEXT,
    restricted INTEGER,
    generated INTEGER
  )`
  
  tables["Convos"] = `CREATE TABLE IF NOT EXISTS Convos(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    channel INTEGER,
    creator VARCHAR(255),
    date INTEGER,
    FOREIGN KEY(channel)
      REFERENCES Channels(id) ON DELETE CASCADE
  )`

  tables["Chats"] = `CREATE TABLE IF NOT EXISTS Chats(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip VARCHAR(255),
    name VARCHAR(255),
    trip VARCHAR(255),
    country VARCHAR(255),
    message TEXT,
    count INTEGER,
    chat_date INTEGER,
    file_path TEXT,
    file_name TEXT,
    file_preview TEXT,
    file_size INTEGER,
    file_dimensions TEXT,
    convo INTEGER,
    channel INTEGER,
    FOREIGN KEY(convo)
      REFERENCES Convos(id) ON DELETE CASCADE,
    FOREIGN KEY(channel)
      REFERENCES Channels(id) ON DELETE CASCADE,
    UNIQUE(count, channel) ON CONFLICT REPLACE
  )`

  tables["Users"] = `CREATE TABLE IF NOT EXISTS Users(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255),
    password VARCHAR(255),
    salt VARCHAR(255),
    session VARCHAR(255),
    date INTEGER,
    identifiers TEXT
  )`

  tables["UserAttributes"] = `CREATE TABLE IF NOT EXISTS UserAttributes(
    user_id INTEGER,
    name VARCHAR(255),
    value TEXT,
    FOREIGN KEY(user_id) REFERENCES Users(id) ON DELETE CASCADE
    )`
  
  tables["GlobalBans"] = `CREATE TABLE IF NOT EXISTS GlobalBans(
    ip VARCHAR(255),
    offense TEXT,
    date INTEGER,
    expiration INTEGER
  )`

  tables["Bans"] = `CREATE TABLE IF NOT EXISTS Bans(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip VARCHAR(255),
    offense TEXT,
    date INTEGER,
    expiration INTEGER,
    banner INTEGER,
    FOREIGN KEY(banner)
      REFERENCES Users(id) ON DELETE CASCADE
  )`

  tables["Owners"] = `CREATE TABLE IF NOT EXISTS Owners(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user INTEGER,
    channel INTEGER,
    permissions INTEGER,
    FOREIGN KEY(user)
      REFERENCES Users(id) ON DELETE CASCADE,
    FOREIGN KEY(channel)
      REFERENCES Channels(id) ON DELETE CASCADE
  )`

  for table := range(tables) {
    query := tables[table]
    err = createTable(db, table, query)
    if err != nil {
      log.Fatal("did not create table", table)
    }
  }
  
  return db
}

