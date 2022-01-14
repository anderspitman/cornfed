package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		tokenCookie, err := r.Cookie("access_token")
		if err != nil {
			sendLoginPage(w, r)
			return
		}

		tokenData, err := db.GetTokenData(tokenCookie.Value)
		if err != nil {
			sendLoginPage(w, r)
			return
		}

		user, err := db.GetUserById(tokenData.UserId)
		if err != nil {
			w.WriteHeader(500)
			io.WriteString(w, err.Error())
			return
		}

		url := fmt.Sprintf("/feeds/%s", user.Email)
		http.Redirect(w, r, url, 303)
	})

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

		user, err := db.GetUserByEmail(email)
		if err != nil {
			err = db.AddUser(email)
			if err != nil {
				w.WriteHeader(500)
				io.WriteString(w, err.Error())
				return
			}
		}

		err = db.AddToken(user.Id, token)
		if err != nil {
			w.WriteHeader(500)
			io.WriteString(w, err.Error())
			return
		}

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

	http.HandleFunc("/feeds/", func(w http.ResponseWriter, r *http.Request) {

		tokenCookie, err := r.Cookie("access_token")
		if err != nil {
			sendLoginPage(w, r)
			return
		}
		tokenData, err := db.GetTokenData(tokenCookie.Value)
		if err != nil {
			sendLoginPage(w, r)
			return
		}

		r.ParseForm()

		if r.Method == "POST" {
			userIdParam := r.Form.Get("user-id")
			feedIdParam := r.Form.Get("feed-id")
			if userIdParam != "" {
				feedName := r.Form.Get("feed-name")
				if feedName == "" {
					w.WriteHeader(400)
					io.WriteString(w, "Blank feed-name")
					return
				}

				userId, err := strconv.Atoi(userIdParam)
				if err != nil {
					w.WriteHeader(400)
					io.WriteString(w, "Invalid user-id param")
					return
				}

				if userId != tokenData.UserId {
					w.WriteHeader(403)
					io.WriteString(w, "Unauthorized")
					return
				}

				_, err = db.GetFeed(userId, feedName)
				if err == nil {
					w.WriteHeader(400)
					io.WriteString(w, "Feed exists")
					return
				}

				err = db.AddFeed(userId, feedName)
				if err != nil {
					w.WriteHeader(400)
					io.WriteString(w, err.Error())
					return
				}
			} else if feedIdParam != "" {
				feedId, err := strconv.Atoi(feedIdParam)
				if err != nil {
					w.WriteHeader(400)
					io.WriteString(w, "Invalid feed-id param")
					return
				}

				feed, err := db.GetFeedById(feedId)
				if err != nil {
					w.WriteHeader(400)
					io.WriteString(w, err.Error())
					return
				}

				if feed.UserId != tokenData.UserId {
					w.WriteHeader(403)
					io.WriteString(w, "Unauthorized")
					return
				}

				user, err := db.GetUserById(tokenData.UserId)
				if err != nil {
					w.WriteHeader(500)
					io.WriteString(w, err.Error())
					return
				}

				err = db.AddSubfeed(feedId, r.Form.Get("subfeed-url"))
				if err != nil {
					w.WriteHeader(500)
					io.WriteString(w, err.Error())
					return
				}

				url := fmt.Sprintf("/feeds/%s/%s", user.Email, feed.Name)
				http.Redirect(w, r, url, 303)
			} else {
				w.WriteHeader(400)
				io.WriteString(w, "Invalid /feeds POST")
				return
			}

			return
		}

		pathParts := strings.Split(r.URL.Path, "/")

		switch len(pathParts) {
		case 3:
			user, err := db.GetUserById(tokenData.UserId)
			if err != nil {
				w.WriteHeader(500)
				io.WriteString(w, err.Error())
				return
			}

			feeds, err := db.GetFeedsByUserId(tokenData.UserId)
			if err != nil {
				w.WriteHeader(500)
				io.WriteString(w, err.Error())
				return
			}

			data := struct {
				Email  string
				UserId int
				Feeds  []*Feed
			}{
				Email:  user.Email,
				UserId: tokenData.UserId,
				Feeds:  feeds,
			}

			err = tmpl.ExecuteTemplate(w, "feeds.tmpl", data)
			if err != nil {
				w.WriteHeader(400)
				io.WriteString(w, err.Error())
				return
			}
		case 4:
			feedName := pathParts[3]

			feed, err := db.GetFeed(tokenData.UserId, feedName)
			if err != nil {
				w.WriteHeader(400)
				io.WriteString(w, err.Error())
				return
			}

			subfeeds, err := db.GetSubfeedsByFeedId(feed.Id)
			if err != nil {
				w.WriteHeader(400)
				io.WriteString(w, err.Error())
				return
			}

			var items []*gofeed.Item

			// TODO: fetch in parallel. Also cache...
			for _, subfeed := range subfeeds {
				inFeed, err := fp.ParseURL(subfeed.Url)
				if err != nil {
					w.WriteHeader(500)
					io.WriteString(w, err.Error())
					return
				}

				for _, item := range inFeed.Items {
					if item.Author == nil {
						if inFeed.Author == nil {
							item.Author = &gofeed.Person{Name: "Unknown Author"}
						} else {
							item.Author = inFeed.Author
						}
					}

					item.Published = item.PublishedParsed.Format(time.RFC3339)
				}

				items = append(items, inFeed.Items...)
			}

			sort.Sort(ByDate(items))

			data := struct {
				FeedId    int
				FeedItems []*gofeed.Item
			}{
				FeedId:    feed.Id,
				FeedItems: items,
			}

			err = tmpl.ExecuteTemplate(w, "feed.tmpl", data)
			if err != nil {
				w.WriteHeader(400)
				io.WriteString(w, err.Error())
				return
			}
		}
	})

	http.HandleFunc("/feeds/todo", func(w http.ResponseWriter, r *http.Request) {

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

func sendLoginPage(w http.ResponseWriter, r *http.Request) {

	host := getHost(r)

	curUrl := fmt.Sprintf("https://%s%s", host, r.RequestURI)

	cookie := &http.Cookie{
		Name:     "return_page",
		Value:    curUrl,
		Secure:   true,
		HttpOnly: true,
		MaxAge:   86400,
		Path:     "/",
		//SameSite: http.SameSiteLaxMode,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/login", 303)
}

func getHost(r *http.Request) string {
	r.ParseForm()
	host := r.Header.Get("X-Forwarded-Host")

	if host == "" {
		host = r.Host
	}

	return host
}

type ByDate []*gofeed.Item

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDate) Less(i, j int) bool { return a[i].PublishedParsed.After(*a[j].PublishedParsed) }
