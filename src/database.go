package main

import (
  "database/sql"
  "log"
)

// interface for database
type Database interface {

  // create all our tables
  // fatal if it doesn't work
  CreateTables()
  
  // aquire underlying sql connection
  // returns nil on error
  Conn() *sql.DB
  // check if we have a channel
  HasChannel(chnl string) (bool, error)
  // check if we have a convo in a channel
  HasConvo(chnl, convo string) (bool, error)
  // get all channels as list
  GetChannels() ([]string, error)
  // get the convos for a channel
  GetConvos(chnlName string) ([]string, error)
  // get top N channels
  GetTopChannels(limit int) ([]string, error)
  // create a convo
  NewConvo(chnl, convo string) error
  // create a new channel
  NewChannel(chnl string) error
  // get channel scrollback for a convo, return the last N chats
  GetScrollback(chnl, convo string, limit int) ([]Chat , error)
  // insert a chat
  // does any rollover
  InsertChat(chat Chat) error
  // deletes a chat
  DeleteChat(chat Chat) error
  // ban the user of a chat
  BanChatUser(chat Chat) error
  // delete all posts from a user of a chat
  NukeChatUser(chat Chat) error
  // ban all tor exits we know of
  BanTor() error
}


func makeDatabase(dbtype, dburl string) Database {
  if dbtype == "postgres" {
    return postgresDatabase{url: dburl}
  }
  /* else if dbtype == "sqlite3" || dbtype == "sqlite" {
    return sqliteDatabase{url: dburl}
  }
  */
  log.Fatalf("non supported database backend: %s", dbtype)
}
