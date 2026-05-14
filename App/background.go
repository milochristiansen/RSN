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
	"strings"
	"time"

	"github.com/milochristiansen/sessionlogger"
	"github.com/mmcdole/gofeed"
)

var fp = gofeed.NewParser()

// Background loops forever doing feed updates and sending push notifications roughly once per minute (plus however long it takes to actually do the update).
func Background() {
	for {
		l := sessionlogger.NewSessionLogger("background-update")
		l.I.Println("Starting update cycle.")

		feeds := GetAllFeeds(l)
		if feeds == nil {
			continue
		}

		// [userID][feedID]{data for feed}
		collation := map[string]map[string]*FeedPushData{}

		for _, data := range feeds {
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
				} else {
					UpdateFeedErrorState(l, feed, 1000)
				}
				continue
			}

			UpdateFeedErrorState(l, feed, 200)

			subs := FeedListSubs(l, feed)
			if len(subs) == 0 {
				l.W.Printf("Feed %v has no subscribers, deleting\n", feed)
				FeedDelete(l, feed)
				continue
			}

			for _, item := range f.Items {
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

				title := item.Title
				if strings.HasPrefix(title, f.Title+" - ") {
					title = strings.TrimPrefix(title, f.Title+" - ")
				}

				ArticleAdd(l, feed, title, item.Link, *t)

				for _, sub := range subs {
					userID, feedName := sub[0], sub[1]

			if collation[userID] == nil {
					collation[userID] = map[string]*FeedPushData{}
				}

				entry, exists := collation[userID][feed]
				if !exists {
					entry = &FeedPushData{
						FeedName: feedName,
					}
					collation[userID][feed] = entry
				}

				entry.ArticleCount++
				if entry.FirstArticleTitle == "" {
					entry.FirstArticleTitle = title
				}
				}
			}
		}

		userFeeds := map[string][]*FeedPushData{}
		for userID, feedMap := range collation {
			feeds := make([]*FeedPushData, 0, len(feedMap))
			for _, entry := range feedMap {
				feeds = append(feeds, entry)
			}
			userFeeds[userID] = feeds
		}

		if len(userFeeds) > 0 {
			go sendNotifications(l, userFeeds)
		}

		time.Sleep(1 * time.Minute)
	}
}

// sendNotifications sends push notifications for new content found during the last background update.
func sendNotifications(l *sessionlogger.Logger, userFeeds map[string][]*FeedPushData) {
	numWorkers := 10
	jobs := make(chan struct {
		userID string
		feeds  []*FeedPushData
	}, len(userFeeds))

	for w := 0; w < numWorkers; w++ {
		go func() {
			for job := range jobs {
				if err := SendPushNotification(l, job.userID, job.feeds); err != nil {
					l.E.Printf("Failed to send push notification to %v: %v\n", job.userID, err)
				}
			}
		}()
	}

	for userID, feeds := range userFeeds {
		jobs <- struct {
			userID string
			feeds  []*FeedPushData
		}{userID, feeds}
	}
	close(jobs)
}
