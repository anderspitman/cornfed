package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/smtp"
	"sync"
	"time"
)

type SmtpConfig struct {
	Server   string `json:"server,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Port     int    `json:"port,omitempty"`
	Sender   string `json:"sender,omitempty"`
}

type Auth struct {
	config              *SmtpConfig
	pendingAuthRequests map[string]*PendingAuthRequest
	mut                 *sync.Mutex
}

type AuthRequest struct {
	Type  string `json:"type"`
	Email string `json:"email"`
}

type PendingAuthRequest struct {
	email string
	code  string
}

func NewAuth(config *SmtpConfig) *Auth {

	pendingAuthRequests := make(map[string]*PendingAuthRequest)
	mut := &sync.Mutex{}

	return &Auth{config, pendingAuthRequests, mut}
}

func (a *Auth) StartEmailValidation(email string) (string, error) {

	requestId, err := genRandomKey()
	if err != nil {
		return "", err
	}

	code, err := genCode()
	if err != nil {
		return "", err
	}

	bodyTemplate := "From: %s <%s>\r\n" +
		"To: %s\r\n" +
		"Subject: Email Validation\r\n" +
		"\r\n" +
		"This is an email validation request from %s. Use the following code to prove you own %s:\r\n" +
		"\r\n" +
		"%s\r\n"

	fromText := "cornfed email validator"
	fromEmail := a.config.Sender
	emailBody := fmt.Sprintf(bodyTemplate, fromText, fromEmail, email, "cornfed", email, code)

	emailAuth := smtp.PlainAuth("", a.config.Username, a.config.Password, a.config.Server)
	srv := fmt.Sprintf("%s:%d", a.config.Server, a.config.Port)
	msg := []byte(emailBody)
	err = smtp.SendMail(srv, emailAuth, fromEmail, []string{email}, msg)
	if err != nil {
		return "", err
	}

	a.mut.Lock()
	a.pendingAuthRequests[requestId] = &PendingAuthRequest{
		email: email,
		code:  code,
	}
	a.mut.Unlock()

	// Requests expire after a certain time
	go func() {
		time.Sleep(60 * time.Second)
		a.mut.Lock()
		delete(a.pendingAuthRequests, requestId)
		a.mut.Unlock()
	}()

	return requestId, nil
}

func (a *Auth) CompleteEmailValidation(requestId, code string) (string, string, error) {

	a.mut.Lock()
	req, exists := a.pendingAuthRequests[requestId]
	delete(a.pendingAuthRequests, requestId)
	a.mut.Unlock()

	if exists && req.code == code {
		token, err := genRandomKey()
		if err != nil {
			return "", "", err
		}
		return token, req.email, nil
	}

	return "", "", errors.New("Failed email validation")
}

func genCode() (string, error) {
	const chars string = "0123456789"
	id := ""
	for i := 0; i < 6; i++ {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		id += string(chars[randIndex.Int64()])
	}
	return id, nil
}

func genRandomKey() (string, error) {
	const chars string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	id := ""
	for i := 0; i < 32; i++ {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		id += string(chars[randIndex.Int64()])
	}
	return id, nil
}
