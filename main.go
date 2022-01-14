package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
)

func main() {
	feedUrl := flag.String("feed-url", "", "Feed URL")
	flag.Parse()

	fp := gofeed.NewParser()

	inFeed, err := fp.ParseURL(*feedUrl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, item := range inFeed.Items {
		fmt.Println(item.Title)
	}

	fmt.Println(inFeed.UpdatedParsed)

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

	atom, err := outFeed.ToAtom()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(atom)
}
