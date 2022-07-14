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

import "time"
import "net/http"

import "github.com/teris-io/shortid"

import "github.com/milochristiansen/sessionlogger"

// Background Updates
// =====================================================================================================================

func GetAllFeeds(l *sessionlogger.Logger) [][2]string {
	rows, err := Queries["GetAllFeeds"].Preped.Query()
	if err != nil {
		l.E.Printf("Feed list failed for background update, error: %v\n", err)
		return nil
	}
	defer rows.Close()

	feeds := [][2]string{}
	for rows.Next() {
		f, id := "", ""
		err := rows.Scan(&f, &id)
		if err != nil {
			l.E.Printf("Feed list failed for background update, error: %v\n", err)
			return nil
		}
		feeds = append(feeds, [2]string{f, id})
	}
	return feeds
}

func ArticleExists(l *sessionlogger.Logger, url string) (exists, ok bool) {
	article := ""
	err := Queries["ArticleExistsByURL"].Preped.QueryRow(url).Scan(&article)
	if err != nil {
		l.E.Printf("DB existence check failed for new article %v, error: %v\n", url, err)
		return false, false
	}
	return article != "", true
}

var articleIDService <-chan string

func init() {
	go func() {
		c := make(chan string)
		articleIDService = c

		idsource := shortid.MustNew(5, shortid.DefaultABC, uint64(time.Now().UnixNano()))

		for {
			c <- idsource.MustGenerate()
		}
	}()
}

func ArticleAdd(l *sessionlogger.Logger, feed, title, url string, published time.Time) {
	article := <-articleIDService
	_, err := Queries["ArticleAdd"].Preped.Exec(article, feed, title, url, published.Unix())
	if err != nil {
		l.E.Printf("Cannot insert article %v into db, error: %v\n", url, err)
	}
}

func FeedListSubs(l *sessionlogger.Logger, feed string) []string {
	rows, err := Queries["FeedListSubs"].Preped.Query(feed)
	if err != nil {
		l.E.Printf("Feed subscribed user list failed for background update, error: %v\n", err)
		return nil
	}
	defer rows.Close()

	users := []string{}
	for rows.Next() {
		user := ""
		err := rows.Scan(&user)
		if err != nil {
			l.E.Printf("Feed subscribed user list failed for background update, error: %v\n", err)
			return nil
		}
		users = append(users, user)
	}
	return users
}

// /api/feed/list
// =====================================================================================================================

type Feed struct {
	ID     string
	Name   string
	URL    string
	Paused bool
}

func FeedList(l *sessionlogger.Logger, id string) []*Feed {
	rows, err := Queries["FeedList"].Preped.Query(id)
	if err != nil {
		l.E.Printf("Feed list failed for user %v, error: %v\n", id, err)
		return nil
	}
	defer rows.Close()

	feeds := []*Feed{}
	for rows.Next() {
		f := &Feed{}
		err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.Paused)
		if err != nil {
			l.E.Printf("Feed list failed for user %v, error: %v\n", id, err)
			return nil
		}
		feeds = append(feeds, f)
	}
	return feeds
}

// /api/feed/details (one row)
// =====================================================================================================================

func FeedDetails(l *sessionlogger.Logger, user, feed string) *Feed {
	f := &Feed{}
	err := Queries["FeedDetails"].Preped.QueryRow(user, feed).Scan(&f.ID, &f.Name, &f.URL, &f.Paused)
	if err != nil {
		l.W.Printf("Error reading feed %v for user %v, error: %v\n", feed, user, err)
		return nil
	}
	return f
}

// /api/feed/articles
// =====================================================================================================================

type Article struct {
	ID        string
	Title     string
	URL       string
	Published time.Time
	Read      bool
}

func FeedArticles(l *sessionlogger.Logger, user, feed string) []*Article {
	rows, err := Queries["FeedArticles"].Preped.Query(user, feed)
	if err != nil {
		l.E.Printf("Feed article list failed for feed %v, user %v. Error: %v\n", feed, user, err)
		return nil
	}
	defer rows.Close()

	articles := []*Article{}
	for rows.Next() {
		a := &Article{}
		var stamp int64
		err := rows.Scan(&a.ID, &a.Title, &a.URL, &stamp, &a.Read)
		if err != nil {
			l.E.Printf("Feed article list failed for feed %v, user %v. Error: %v\n", feed, user, err)
			return nil
		}
		a.Published = time.Unix(stamp, 0)
		articles = append(articles, a)
	}
	return articles
}

// /api/feed/subscribe
// =====================================================================================================================

var feedIDService <-chan string

func init() {
	go func() {
		c := make(chan string)
		feedIDService = c

		idsource := shortid.MustNew(7, shortid.DefaultABC, uint64(time.Now().UnixNano()))

		for {
			c <- idsource.MustGenerate()
		}
	}()
}

type FeedSubscribeData struct {
	URL  string
	Name string
}

func FeedSubscribe(l *sessionlogger.Logger, id, url, name string) int {
	// First things first: Check to see if a feed with this url already esists.
	feed := ""
	err := Queries["FeedExistsByURL"].Preped.QueryRow(url).Scan(&feed)
	if err != nil {
		l.E.Printf("DB existence check failed for new feed %v, error: %v\n", url, err)
		return http.StatusInternalServerError
	}
	if feed == "" {
		// Create new feed.
		feed = <-feedIDService
		_, err = Queries["FeedAdd"].Preped.Exec(feed, url)
		if err != nil {
			l.E.Printf("Cannot insert feed %v into db, error: %v\n", url, err)
			return http.StatusInternalServerError
		}
	}

	ok := 0
	err = Queries["FeedAlreadySubscibed"].Preped.QueryRow(id, feed).Scan(&ok)
	if err != nil {
		l.E.Printf("DB existence check failed for subscribed feed %v by user %v, error: %v\n", feed, id, err)
		return http.StatusInternalServerError
	}
	if ok == 1 {
		l.W.Printf("Feed %v already subscribed by user %v.\n", feed, id)
		// This isn't a straight up error, but it isn't OK either.
		return http.StatusAccepted
	}

	_, err = Queries["FeedSubscibe"].Preped.Exec(id, feed, name)
	if err != nil {
		l.E.Printf("Failed subscribing feed %v as user %v, error: %v\n", feed, id, err)
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

// /api/feed/unsubscribe
// =====================================================================================================================

func FeedUnsub(l *sessionlogger.Logger, user, feed string) int {
	_, err := Queries["FeedUnsub1"].Preped.Exec(user, feed)
	if err != nil {
		l.E.Printf("Failed unsubscribing feed %v as user %v, error: %v\n", feed, user, err)
		return http.StatusInternalServerError
	}

	// Now check if the feed has no subscribers.
	hassub := 0
	err = Queries["FeedHasSubs"].Preped.QueryRow(feed).Scan(&hassub)
	if err != nil {
		l.E.Printf("Feed subscriber check failed for feed %v, error: %v\n", feed, err)
		return http.StatusInternalServerError
	}
	if hassub == 1 {
		// If the feed still has other subscribers delete our paused flags and slink off into the night.
		_, err = Queries["FeedUnsub2"].Preped.Exec(user, feed)
		if err != nil {
			l.E.Printf("Failed unsubscribing feed %v as user %v, error: %v\n", feed, user, err)
			return http.StatusInternalServerError
		}
		return http.StatusOK
	}

	// No subscribers left, delete feed for real.
	_, err = Queries["FeedDelete"].Preped.Exec(feed)
	if err != nil {
		l.E.Printf("Failed deleting feed %v, error: %v\n", feed, err)
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

// /api/feed/pause
// =====================================================================================================================

func FeedPause(l *sessionlogger.Logger, user, feed string) int {
	_, err := Queries["FeedPause"].Preped.Exec(user, feed)
	if err != nil {
		l.E.Printf("Failed pausing feed %v, error: %v\n", feed, err)
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

// //api/feed/unpause
// =====================================================================================================================

func FeedUnpause(l *sessionlogger.Logger, user, feed string) int {
	_, err := Queries["FeedUnpause"].Preped.Exec(user, feed)
	if err != nil {
		l.E.Printf("Failed unpausing feed %v, error: %v\n", feed, err)
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

// /api/article/read
// =====================================================================================================================

func ArticleMarkRead(l *sessionlogger.Logger, user, article string) int {
	_, err := Queries["ArticleRead"].Preped.Exec(user, article)
	if err != nil {
		l.E.Printf("Failed marking article (%v) read, error: %v\n", article, err)
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

// /api/article/unread
// =====================================================================================================================

func ArticleMarkUnread(l *sessionlogger.Logger, user, article string) int {
	_, err := Queries["ArticleUnread"].Preped.Exec(user, article)
	if err != nil {
		l.E.Printf("Failed marking article (%v) unread, error: %v\n", article, err)
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

// /api/getunread
// =====================================================================================================================

type UnreadArticle struct {
	ID        string
	Title     string
	URL       string
	FeedName  string // Feed *name*, not ID.
	FeedID    string
	Published time.Time
}

func GetUnread(l *sessionlogger.Logger, user string) []*UnreadArticle {
	rows, err := Queries["GetUnread"].Preped.Query(user)
	if err != nil {
		l.E.Printf("Unread article list failed for user %v. Error: %v\n", user, err)
		return nil
	}
	defer rows.Close()

	articles := []*UnreadArticle{}
	for rows.Next() {
		a := &UnreadArticle{}
		var stamp int64
		err := rows.Scan(&a.ID, &a.Title, &a.URL, &a.FeedName, &a.FeedID, &stamp)
		if err != nil {
			l.E.Printf("Unread article list failed for user %v. Error: %v\n", user, err)
			return nil
		}
		a.Published = time.Unix(stamp, 0)
		articles = append(articles, a)
	}
	return articles
}

// /api/recentread
// =====================================================================================================================

// Same as UnreadArticle, but with the addition of a time when it was added to the read flags.
type ReadArticle struct {
	ID        string
	Title     string
	URL       string
	FeedName  string
	FeedID    string
	Published time.Time
	ReadAt    time.Time
}

func GetRecentRead(l *sessionlogger.Logger, user string) []*ReadArticle {
	rows, err := Queries["GetRecentRead"].Preped.Query(user)
	if err != nil {
		l.E.Printf("Recently read article list failed for user %v. Error: %v\n", user, err)
		return nil
	}
	defer rows.Close()

	articles := []*ReadArticle{}
	for rows.Next() {
		a := &ReadArticle{}
		var stampA, stampB int64
		err := rows.Scan(&a.ID, &a.Title, &a.URL, &a.FeedName, &a.FeedID, &stampA, &stampB)
		if err != nil {
			l.E.Printf("Recently read article list failed for user %v. Error: %v\n", user, err)
			return nil
		}
		a.Published = time.Unix(stampA, 0)
		a.ReadAt = time.Unix(stampB, 0)
		articles = append(articles, a)
	}
	return articles
}