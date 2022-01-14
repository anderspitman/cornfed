package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
)

//go:embed templates
var fs embed.FS

func main() {
	smtpServer := flag.String("smtp-server", "", "SMTP Server")
	smtpUsername := flag.String("smtp-username", "", "SMTP Username")
	smtpPassword := flag.String("smtp-password", "", "SMTP Password")
	smtpSender := flag.String("smtp-sender", "", "SMTP Sender")
	flag.Parse()

	db := NewDatabase()
	fmt.Println(db)

	config := &SmtpConfig{
		Server:   *smtpServer,
		Username: *smtpUsername,
		Password: *smtpPassword,
		Sender:   *smtpSender,
		Port:     587,
	}

	auth := NewAuth(config)

	fp := gofeed.NewParser()

	tmpl, err := template.ParseFS(fs, "templates/*.tmpl")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		if r.Method != "GET" {
			w.WriteHeader(405)
			io.WriteString(w, "Invalid method")
			return
		}

		data := struct{}{}
		err := tmpl.ExecuteTemplate(w, "login.tmpl", data)
		if err != nil {
			w.WriteHeader(400)
			io.WriteString(w, err.Error())
			return
		}
	})

	http.HandleFunc("/complete-login", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		if r.Method != "POST" {
			w.WriteHeader(405)
			io.WriteString(w, "Invalid method")
			return
		}

		email := r.Form.Get("email")
		if email == "" {
			w.WriteHeader(400)
			io.WriteString(w, "email param missing")
			return
		}

		requestId, err := auth.StartEmailValidation(email)
		if err != nil {
			w.WriteHeader(400)
			io.WriteString(w, err.Error())
			return
		}

		data := struct {
			RequestId string
		}{
			RequestId: requestId,
		}

		err = tmpl.ExecuteTemplate(w, "complete-login.tmpl", data)
		if err != nil {
			w.WriteHeader(400)
			io.WriteString(w, err.Error())
			return
		}
	})

	http.HandleFunc("/complete-email-validation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			io.WriteString(w, "Invalid method")
			return
		}

		r.ParseForm()

		requestId := r.Form.Get("request-id")
		if requestId == "" {
			w.WriteHeader(400)
			io.WriteString(w, "request-id param missing")
			return
		}

		code := r.Form.Get("code")
		if requestId == "" {
			w.WriteHeader(400)
			io.WriteString(w, "request-id param missing")
			return
		}

		token, email, err := auth.CompleteEmailValidation(requestId, code)
		if err != nil {
			w.WriteHeader(400)
			io.WriteString(w, err.Error())
			return
		}

		fmt.Println("validated", email)

		//db.AddToken(token, email)

		cookie := &http.Cookie{
			Name:     "access_token",
			Value:    token,
			Secure:   true,
			HttpOnly: true,
			MaxAge:   86400 * 365,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			//SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(w, cookie)

		returnPageCookie, err := r.Cookie("return_page")
		if err != nil {
			http.Redirect(w, r, "/", 303)
		} else {
			http.Redirect(w, r, returnPageCookie.Value, 303)
		}

	})

	http.HandleFunc("/feeds", func(w http.ResponseWriter, r *http.Request) {

		pathParts := strings.Split(r.URL.Path, "/")

		if len(pathParts) < 2 {
			w.WriteHeader(400)
			io.WriteString(w, "Invalid URL path")
			return
		}

		format := pathParts[1]

		var feedUrl string
		if format == "rss" || format == "json" || format == "atom" {
			feedUrl = "https://" + strings.Join(pathParts[2:], "/")
		} else {
			feedUrl = "https://" + r.URL.Path[1:]
		}

		fmt.Println(feedUrl)

		inFeed, err := fp.ParseURL(feedUrl)
		if err != nil {
			w.WriteHeader(500)
			io.WriteString(w, err.Error())
			return
		}

		outFeed, err := convert(inFeed)
		if err != nil {
			w.WriteHeader(500)
			io.WriteString(w, err.Error())
			return
		}

		var out string

		switch format {
		case "rss":
			out, err = outFeed.ToRss()
		case "json":
			out, err = outFeed.ToJSON()
		case "atom":
			fallthrough
		default:
			out, err = outFeed.ToAtom()
		}

		if err != nil {
			w.WriteHeader(500)
			io.WriteString(w, err.Error())
			return
		}

		w.Write([]byte(out))
	})

	http.ListenAndServe(":9004", nil)
}

func convert(inFeed *gofeed.Feed) (*feeds.Feed, error) {

	outFeed := &feeds.Feed{
		Title:       inFeed.Title,
		Link:        &feeds.Link{Href: inFeed.Link},
		Description: inFeed.Description,
		Items:       []*feeds.Item{},
	}

	if inFeed.PublishedParsed != nil {
		outFeed.Created = *inFeed.PublishedParsed
	}
	if inFeed.UpdatedParsed != nil {
		outFeed.Updated = *inFeed.UpdatedParsed
	}

	for _, inItem := range inFeed.Items {
		outItem := &feeds.Item{
			Title:       inItem.Title,
			Link:        &feeds.Link{Href: inItem.Link},
			Description: inItem.Description,
			Content:     inItem.Content,
		}

		if inItem.PublishedParsed != nil {
			outItem.Created = *inItem.PublishedParsed
		}
		if inItem.UpdatedParsed != nil {
			outItem.Updated = *inItem.UpdatedParsed
		}
		if inItem.Author != nil {
			outItem.Author = &feeds.Author{Name: inItem.Author.Name, Email: inItem.Author.Email}
		}

		outFeed.Items = append(outFeed.Items, outItem)
	}

	return outFeed, nil
}
