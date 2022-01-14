package main

import (
	//"flag"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
)

func main() {
	//feedUrl := flag.String("feed-url", "", "Feed URL")
	//flag.Parse()

	fp := gofeed.NewParser()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		feedUrl := "https://" + r.URL.Path[1:]
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

		atom, err := outFeed.ToAtom()
		if err != nil {
			w.WriteHeader(500)
			io.WriteString(w, err.Error())
			return
		}

		w.Write([]byte(atom))
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
