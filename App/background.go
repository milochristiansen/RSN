/*
Copyright 2020-2022 by Milo Christiansen

This software is provided 'as-is', without any express or implied warranty. In
no event will the authors be held liable for any damages arising from the use of
this software.

Permission is granted to anyone to use this software for any purpose, including
commercial applications, and to alter it and redistribute it freely, subject to
the following restrictions:

1. The origin of this software must not be misrepresented; you must not claim
that you wrote the original software. If you use this software in a product, an
acknowledgment in the product documentation would be appreciated but is not
required.

2. Altered source versions must be plainly marked as such, and must not be
misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.
*/

package main

import (
	"errors"
	"time"

	"github.com/milochristiansen/sessionlogger"
	"github.com/mmcdole/gofeed"
)

var fp = gofeed.NewParser()

func Background() {
	l := sessionlogger.NewMasterLogger()
	l.I.Println("Starting background process.")

	// There are a lot of places in here where an error can occur, but we simply log+ignore
	for {
		l.I.Println("Starting update cycle.")

		// updated := map[string]bool{}

		// For every single feed in the DB
		feeds := GetAllFeeds(l)
		if feeds == nil {
			continue
		}
		for _, data := range feeds {
			// Check if there are new items
			url, feed := data[0], data[1]

			f, err := fp.ParseURL(url)
			if err != nil {
				l.E.Printf("Error loading feed %v (%v), error: %v\n", feed, url, err)
				if f != nil {
					l.I.Printf("%#v\n", f)
				}

				hterr := &gofeed.HTTPError{}
				if errors.As(err, hterr) {
					UpdateFeedErrorState(l, feed, hterr.StatusCode)
				}
				continue
			}

			UpdateFeedErrorState(l, feed, 200) // May not actually be a 200, but it will be a 200 class.

			for _, item := range f.Items {
				// Check if we know of the item

				exists, ok := ArticleExists(l, item.Link)
				if !ok || exists {
					continue
				}

				t := item.PublishedParsed
				if t == nil {
					t = item.UpdatedParsed
					if t == nil {
						t2 := time.Now()
						t = &t2
					}
				}

				ArticleAdd(l, feed, item.Title, item.Link, *t)

				// users := FeedListSubs(l, feed)
				// if users == nil {
				// 	continue
				// }
				// for _, user := range users {
				// 	updated[user] = true
				// }
			}
		}

		time.Sleep(1 * time.Minute)
	}
}
