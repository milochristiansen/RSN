/*
Copyright 2020-2021 by Milo Christiansen

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

// RSN2: Multi-user RSS feed tracker.
package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/milochristiansen/sessionlogger"
)

const MaxBodyBytes = int64(65536)

func main() {
	// /api/feed/list
	http.HandleFunc("/api/feed/list", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/feed/list")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		feeds := FeedList(l, user.UID)
		if feeds == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err := json.NewEncoder(w).Encode(feeds)
		if err != nil {
			l.E.Printf("Error encoding payload. Error: %v\n", err)
			return
		}
	})

	// /api/feed/details
	http.HandleFunc("/api/feed/details", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/feed/details")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		feed := r.FormValue("id")
		if feed == "" {
			l.W.Printf("Missing feed ID.\n")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		details := FeedDetails(l, user.UID, feed)
		if details == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err := json.NewEncoder(w).Encode(details)
		if err != nil {
			l.E.Printf("Error encoding payload. Error: %v\n", err)
			return
		}
	})

	// /api/feed/articles
	http.HandleFunc("/api/feed/articles", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/feed/articles")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		feed := r.FormValue("id")
		if feed == "" {
			l.W.Printf("Missing feed ID.\n")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		articles := FeedArticles(l, user.UID, feed)
		if articles == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err := json.NewEncoder(w).Encode(articles)
		if err != nil {
			l.W.Printf("Error encoding payload. Error: %v\n", err)
			return
		}
	})

	// /api/feed/subscribe
	http.HandleFunc("/api/feed/subscribe", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/feed/subscribe")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

		data := &FeedSubscribeData{}
		err := json.NewDecoder(r.Body).Decode(data)
		if err != nil {
			l.W.Printf("Error parsing feed subscribe body. Error: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if data.Name == "" {
			l.W.Printf("No feed name given.\n")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = url.ParseRequestURI(data.URL)
		if err != nil {
			l.W.Printf("Malformed URL. Error: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(FeedSubscribe(l, user.UID, data.URL, data.Name))
	})

	// /api/feed/unsubscribe
	http.HandleFunc("/api/feed/unsubscribe", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/feed/unsubscribe")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		feed := r.FormValue("id")
		if feed == "" {
			l.W.Printf("Missing feed ID.\n")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(FeedUnsub(l, user.UID, feed))
	})

	// /api/feed/pause
	http.HandleFunc("/api/feed/pause", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/feed/pause")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		feed := r.FormValue("id")
		if feed == "" {
			l.W.Printf("Missing feed ID.\n")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(FeedPause(l, user.UID, feed))
	})

	// /api/feed/unpause
	http.HandleFunc("/api/feed/unpause", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/feed/unpause")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		feed := r.FormValue("id")
		if feed == "" {
			l.W.Printf("Missing feed ID.\n")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(FeedUnpause(l, user.UID, feed))
	})

	// /api/article/read
	http.HandleFunc("/api/article/read", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/article/read")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		article := r.FormValue("id")
		if article == "" {
			l.W.Printf("Missing article ID.\n")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(ArticleMarkRead(l, user.UID, article))
	})

	// /api/article/unread
	http.HandleFunc("/api/article/unread", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/article/unread")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		article := r.FormValue("id")
		if article == "" {
			l.W.Printf("Missing article ID.\n")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(ArticleMarkUnread(l, user.UID, article))
	})

	// /api/getunread
	http.HandleFunc("/api/getunread", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/getunread")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		articles := GetUnread(l, user.UID)
		if articles == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err := json.NewEncoder(w).Encode(articles)
		if err != nil {
			l.W.Printf("Error encoding payload. Error: %v\n", err)
			return
		}
	})

	// /api/recentread
	http.HandleFunc("/api/recentread", func(w http.ResponseWriter, r *http.Request) {
		l := sessionlogger.NewSessionLogger("/api/recentread")

		user, status := GetSessionData(l, w, r)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		articles := GetRecentRead(l, user.UID)
		if articles == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err := json.NewEncoder(w).Encode(articles)
		if err != nil {
			l.W.Printf("Error encoding payload. Error: %v\n", err)
			return
		}
	})

	// /auth/login/google
	http.HandleFunc("/auth/login/google", GoogleLoginEndpoint)
	// /auth/redirect/google
	http.HandleFunc("/auth/redirect/google", GoogleRedirectEndpoint)
	// /auth/logout
	http.HandleFunc("/auth/logout", LogoutEndpoint)
	// /auth/whoami
	http.HandleFunc("/auth/whoami", WhoAmIEndpoint)

	go Background()

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}
