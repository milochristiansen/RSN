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

package main

import _ "github.com/mattn/go-sqlite3"
import "database/sql"

var DB *sql.DB

const (
	UserProviderGoogle = iota // Currently the only one.
)

var InitCode = `
create table if not exists Users (
	ID text primary key,

	Provider integer,
	Subject text not null
);

create table if not exists Feeds (
	ID text primary key,

	URL text unique not null,

	HTTPError integer
);
create unique index if not exists FeedURLs on Feeds(URL);

create table if not exists Articles (
	ID text primary key,
	Feed text not null,

	Title text collate nocase,
	URL text unique not null,

	Published integer,

	foreign key (Feed) references Feeds(ID) on delete cascade
);
create unique index if not exists ArticleURLs on Articles(URL);

create table if not exists ReadFlags (
	User text not null,
	Article text not null,
	ReadTime integer not null default (strftime('%s', 'now')),

	foreign key (Article) references Articles(ID) on delete cascade
);

create table if not exists PausedFlags (
	User text not null,
	Feed text not null,

	foreign key (Feed) references Feeds(ID) on delete cascade
);

create table if not exists Subscribed (
	User text not null,
	Feed text not null,
	Name text collate nocase,

	foreign key (Feed) references Feeds(ID) on delete cascade
);
`

var Queries = map[string]*queryHolder{
	// Auth UID lookup and creation.
	"GetUID": {`
		select ID from Users where Provider = ?1 and Subject = ?2;
	`, nil},
	"CreatNewUser": {`
		insert into Users (ID, Provider, Subject) values (?1, ?2, ?3);
	`, nil},

	// Background updater
	"GetAllFeeds": {`
		select URL, ID from Feeds;
	`, nil},
	"ArticleExistsByURL": {`
		select ID from Articles where URL = ?1 union select "" order by 1 desc limit 1;
	`, nil},
	"ArticleAdd": {`
		insert into Articles (ID, Feed, Title, URL, Published) values (?1, ?2, ?3, ?4, ?5);
	`, nil},
	"FeedListSubs": {`
		select User from Subscribed where Feed = ?1;
	`, nil},
	"UpdateFeedErrState": {`
		update Feeds set HTTPError = ?2 where ID = ?1;
	`, nil},

	// /api/feed/list
	"FeedList": {`
		select ID, (
				select Name from Subscribed where Feed = Feeds.ID and User = ?1
			) as Name, URL, (
			ID in (select Feed from PausedFlags where User = ?1)
		), HTTPError from Feeds where (
			ID in (select Feed from Subscribed where User = ?1)
		) order by Name;
	`, nil},
	// /api/feed/details (one row)
	"FeedDetails": {`
		select ID, (
			select Name from Subscribed where Feed = ?2 and User = ?1
		), URL, (
			ID in (select Feed from PausedFlags where User = ?1)
		), HTTPError from Feeds where (
			ID = ?2 and
			ID in (select Feed from Subscribed where User = ?1)
		);
	`, nil},
	// /api/feed/articles
	"FeedArticles": {`
		select ID, Title, URL, Published, (
			ID in (select Article from ReadFlags where User = ?1)
		) from Articles where (
			Feed = ?2 and
			Feed in (select Feed from Subscribed where User = ?1)
		) order by Published;
	`, nil},
	// /api/feed/subscribe
	"FeedExistsByURL": {`
		select ID from Feeds where URL = ?1 union select "" order by 1 desc limit 1;
	`, nil},
	"FeedAdd": {`
		insert into Feeds (ID, URL) values (?1, ?2);
	`, nil},
	"FeedAlreadySubscibed": {`
		select exists(select 1 from Subscribed where User = ?1 and Feed = ?2);
	`, nil},
	"FeedSubscibe": {`
		insert into Subscribed (User, Feed, Name) values (?1, ?2, ?3);
	`, nil},
	// /api/feed/unsubscribe
	"FeedUnsub1": {`
		delete from Subscribed where User = ?1 and Feed = ?2;
	`, nil},
	"FeedUnsub2": {`
		delete from PausedFlags where User = ?1 and Feed = ?2;
	`, nil},
	"FeedHasSubs": {`
		select exists(select 1 from Subscribed where Feed = ?1 limit 1);
	`, nil},
	"FeedDelete": {`
		delete from Feeds where ID = ?1;
	`, nil},
	// /api/feed/pause
	"FeedExists": {`
		select exists(select 1 from Feeds where ID = ?1 limit 1);
	`, nil},
	"FeedPause": {`
		insert into PausedFlags (User, Feed) values (?1, ?2);
	`, nil},
	// /api/feed/unpause
	"FeedUnpause": {`
		delete from PausedFlags where User = ?1 and Feed = ?2;
	`, nil},

	// /api/article/read
	"ArticleRead": {`
		insert into ReadFlags (User, Article) values (?1, ?2);
	`, nil},
	// /api/article/unread
	"ArticleUnread": {`
		delete from ReadFlags where User = ?1 and Article = ?2;
	`, nil},
	// /api/getunread
	"GetUnread": {`
		select a.ID, a.Title, a.URL, fn.Name, fn.Feed, a.Published from Articles a
		inner join Subscribed fn on fn.Feed = a.Feed and fn.User = ?1 where (
			not a.ID in (select Article from ReadFlags where User = ?1) and
			not a.Feed in (select Feed from PausedFlags where User = ?1)
		) order by Published;
	`, nil},
	// /api/recentread
	"GetRecentRead":{`
		select a.ID, a.Title, a.URL, fn.Name, fn.Feed, a.Published, rf.ReadTime from Articles a
		inner join Subscribed fn on fn.Feed = a.Feed and fn.User = ?1
		inner join ReadFlags rf on rf.Article = a.ID and rf.User = ?1 
		order by rf.ReadTime desc limit 25;
	`, nil},
}

func init() {
	var err error
	DB, err = sql.Open("sqlite3", "file:feeds.db?_foreign_keys=true")
	if err != nil {
		panic(err)
	}

	_, err = DB.Exec(InitCode)
	if err != nil {
		panic("Error loading DB init code:\n" + err.Error())
	}

	for _, v := range Queries {
		err := v.Init()
		if err != nil {
			panic("Error loading query: " + v.Code + "\n\n" + err.Error())
		}
	}
}

type queryHolder struct {
	Code   string
	Preped *sql.Stmt
}

func (q *queryHolder) Init() error {
	var err error
	q.Preped, err = DB.Prepare(q.Code)
	return err
}
