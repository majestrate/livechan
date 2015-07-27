//
// sqlite database backend implementation
//
package main

import (
  "database/sql"
)

type sqliteDatabase struct {
  url string
  conn *sql.DB
}
