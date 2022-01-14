package main

import (
	"database/sql"
	//"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type Database struct {
	db *sql.DB
}

type User struct {
	Id    int
	Email string
}

func NewDatabase() *Database {

	db, err := sql.Open("sqlite3", "./cornfed.db")
	if err != nil {
		log.Fatal(err)
	}

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS
        users(id INTEGER NOT NULL PRIMARY KEY, email TEXT UNIQUE);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return nil
	}

	return &Database{
		db: db,
	}
}

func (d *Database) AddUser(email string) error {
	stmt := `
        INSERT INTO users(email) VALUES(?)
        `
	_, err := d.db.Exec(stmt, email)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) GetUserByEmail(email string) (*User, error) {
	stmt := `
        SELECT * FROM users WHERE email=?
        `
	row := d.db.QueryRow(stmt, email)

	user := &User{}
	err := row.Scan(&user.Id, &user.Email)
	if err != nil {
		return nil, err
	}

	return user, nil
}
