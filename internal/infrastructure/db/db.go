package db

import (
	"database/sql"
)

type DB struct {
	*sql.DB
}

func NewDB(db *sql.DB) *DB {
	return &DB{DB: db}
}
