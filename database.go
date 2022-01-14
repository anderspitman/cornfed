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

type Feed struct {
	Id     int
	UserId int
	Name   string
}

type FeedMember struct {
	Id     int
	FeedId int
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

	sqlStmt = `
	CREATE TABLE IF NOT EXISTS feeds(
                id INTEGER NOT NULL PRIMARY KEY,
                user_id INTEGER,
                name TEXT,
                FOREIGN KEY(user_id) REFERENCES users(id)
        );
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return nil
	}

	sqlStmt = `
	CREATE TABLE IF NOT EXISTS feed_members(
                id INTEGER NOT NULL PRIMARY KEY,
                feed_id INTEGER,
                url TEXT,
                FOREIGN KEY(feed_id) REFERENCES feeds(id)
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

func (d *Database) GetUserById(id int) (*User, error) {
	stmt := `
        SELECT * FROM users WHERE id=?
        `
	row := d.db.QueryRow(stmt, id)

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
        SELECT * FROM tokens WHERE token=?;
        `
	row := d.db.QueryRow(stmt, token)

	tokenData := &TokenData{}
	err := row.Scan(&tokenData.UserId, &tokenData.Token)
	if err != nil {
		return nil, err
	}

	return tokenData, nil
}

func (d *Database) GetFeed(userId int, feedName string) (*Feed, error) {
	stmt := `
        SELECT * FROM feeds WHERE user_id=? AND name=?;
        `
	row := d.db.QueryRow(stmt, userId, feedName)

	feed := &Feed{}
	err := row.Scan(&feed.Id, &feed.UserId, &feed.Name)
	if err != nil {
		return nil, err
	}

	return feed, nil
}

func (d *Database) GetFeedsByUserId(userId int) ([]*Feed, error) {
	stmt := `
        SELECT * FROM feeds WHERE user_id=?
        `
	rows, err := d.db.Query(stmt, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*Feed

	for rows.Next() {
		feed := &Feed{}
		if err := rows.Scan(&feed.Id, &feed.UserId, &feed.Name); err != nil {
			return nil, err
		}

		feeds = append(feeds, feed)
	}

	rerr := rows.Close()
	if rerr != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return feeds, nil
}

func (d *Database) AddFeed(userId int, feedName string) error {
	stmt := `
        INSERT INTO feeds(user_id, name) VALUES(?,?)
        `
	_, err := d.db.Exec(stmt, userId, feedName)
	if err != nil {
		return err
	}
	return nil
}
