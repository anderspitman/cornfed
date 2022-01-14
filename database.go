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

type TokenData struct {
	UserId int
	Token  string
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

	// TODO: enable foreign key support
	sqlStmt = `
	CREATE TABLE IF NOT EXISTS tokens(
                user_id INTEGER,
                token TEXT,
                FOREIGN KEY(user_id) REFERENCES users(id)
        );
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

func (d *Database) AddToken(userId int, token string) error {
	stmt := `
        INSERT INTO tokens(user_id, token) VALUES(?,?)
        `
	_, err := d.db.Exec(stmt, userId, token)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) GetTokenData(token string) (*TokenData, error) {
	stmt := `
        SELECT * FROM tokens WHERE token=?
        `
	row := d.db.QueryRow(stmt, token)

	tokenData := &TokenData{}
	err := row.Scan(&tokenData.UserId, &tokenData.Token)
	if err != nil {
		return nil, err
	}

	return tokenData, nil
}
