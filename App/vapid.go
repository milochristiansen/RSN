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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/milochristiansen/sessionlogger"
)

// PushSubscription holds the data for a user's push subscription.
type PushSubscription struct {
	Endpoint string
	P256DH   string
	Auth     string
	UserID   string
}

// PushMessage is the JSON structure sent in push notifications.
type PushMessage struct {
	Ts  int64  `json:"ts"`
	Msg string `json:"msg"`
}

// FeedPushData holds aggregated feed information for push message construction.
type FeedPushData struct {
	FeedName         string
	ArticleCount     int
	FirstArticleTitle string
}

// PushSubscribeRequest holds the data for registering a push subscription.
type PushSubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	P256DH   string `json:"keys"`
	Auth     string `json:"auth"`
}

// constructPushMessage builds a formatted push notification message from aggregated feed data.
func constructPushMessage(feeds []*FeedPushData) string {
	if len(feeds) == 0 {
		return ""
	}

	totalArticles := 0
	for _, f := range feeds {
		totalArticles += f.ArticleCount
	}

	if len(feeds) == 1 {
		f := feeds[0]
		if f.ArticleCount == 1 {
			return fmt.Sprintf("New article: %s from %s", f.FirstArticleTitle, f.FeedName)
		}
		return fmt.Sprintf("%d new articles from %s", f.ArticleCount, f.FeedName)
	}

	feedList := make([]string, 0, len(feeds))
	maxShow := 3
	for i, f := range feeds {
		feedList = append(feedList, f.FeedName)
		if i >= maxShow-1 && i < len(feeds)-1 {
			extra := len(feeds) - maxShow
			feedList = append(feedList, fmt.Sprintf("and %d more", extra))
			break
		}
	}

	return fmt.Sprintf("%d new articles from %s", totalArticles, strings.Join(feedList, ", "))
}

// RegisterPushSubscription stores or updates a push subscription in the database.
func RegisterPushSubscription(userID, endpoint, p256dh, auth string) error {
	// The user may need to update their subscription. To facilitate this first see if their device has a
	// subscription and remove it for replacing.
	// Most likely the subscription does not exist, so we ignore the error if there is one.
	_ = RemovePushSubscription(userID, endpoint)

	_, err := Queries["PushSubInsert"].Preped.Exec(userID, endpoint, p256dh, auth)
	if err != nil {
		return fmt.Errorf("failed to register push subscription: %w", err)
	}
	return nil
}

// GetUserPushSubscriptions retrieves all push subscriptions for a user.
func GetUserPushSubscriptions(userID string) ([]PushSubscription, error) {
	rows, err := Queries["PushSubGetByUser"].Preped.Query(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get push subscriptions for user %v: %w", userID, err)
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var sub PushSubscription
		sub.UserID = userID
		if err := rows.Scan(&sub.Endpoint, &sub.P256DH, &sub.Auth); err != nil {
			return nil, fmt.Errorf("failed to scan push subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// RemovePushSubscription deletes a push subscription from the database.
func RemovePushSubscription(userID, endpoint string) error {
	_, err := Queries["PushSubDelete"].Preped.Exec(userID, endpoint)
	if err != nil {
		return fmt.Errorf("failed to remove push subscription: %w", err)
	}
	return nil
}

// PushSubscriptionExists checks if a push subscription exists for a user and endpoint.
func PushSubscriptionExists(userID, endpoint string) bool {
	var exists int
	err := Queries["PushSubExists"].Preped.QueryRow(userID, endpoint).Scan(&exists)
	if err != nil {
		return false
	}
	return exists == 1
}

// SendTestPushNotification sends a test push notification to all subscriptions for a user.
func SendTestPushNotification(l *sessionlogger.Logger, userID string) error {
	subs, err := GetUserPushSubscriptions(userID)
	if err != nil {
		return fmt.Errorf("failed to get push subscriptions for user %v: %w", userID, err)
	}

	if len(subs) == 0 {
		return fmt.Errorf("no push subscriptions found for user %v", userID)
	}

	msg := "This is a test push notification from RSN"
	payload := PushMessage{Ts: time.Now().Unix(), Msg: msg}
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push message: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}

	for _, sub := range subs {
		resp, err := webpush.SendNotification(message, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256DH,
				Auth:   sub.Auth,
			},
		}, &webpush.Options{
			VAPIDPublicKey:  VAPIDPublicKey,
			VAPIDPrivateKey: VAPIDPrivateKey,
			HTTPClient:      client,
		})
		if err != nil {
			if resp != nil && (resp.StatusCode == 410 || resp.StatusCode == 404) {
				l.W.Printf("Stale subscription for %v, removing\n", sub.Endpoint)
				if err := RemovePushSubscription(userID, sub.Endpoint); err != nil {
					l.E.Printf("Failed to remove stale subscription for %v: %v\n", sub.Endpoint, err)
				}
			} else {
				l.E.Printf("Push notification failed for %v: %v\n", sub.Endpoint, err)
			}
		}
	}
	return nil
}

// SendPushNotification sends a push notification to all subscriptions for a user.
func SendPushNotification(l *sessionlogger.Logger, userID string, feeds []*FeedPushData) error {
	subs, err := GetUserPushSubscriptions(userID)
	if err != nil {
		return fmt.Errorf("failed to get push subscriptions for user %v: %w", userID, err)
	}

	if len(subs) == 0 {
		return nil
	}

	msg := constructPushMessage(feeds)
	if msg == "" {
		return nil
	}

	payload := PushMessage{Ts: time.Now().Unix(), Msg: msg}
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push message: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}

	for _, sub := range subs {
		resp, err := webpush.SendNotification(message, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256DH,
				Auth:   sub.Auth,
			},
		}, &webpush.Options{
			VAPIDPublicKey:  VAPIDPublicKey,
			VAPIDPrivateKey: VAPIDPrivateKey,
			HTTPClient:      client,
		})
		if err != nil {
			if resp != nil && (resp.StatusCode == 410 || resp.StatusCode == 404) {
				l.W.Printf("Stale subscription for %v, removing\n", sub.Endpoint)
				if err := RemovePushSubscription(userID, sub.Endpoint); err != nil {
					l.E.Printf("Failed to remove stale subscription for %v: %v\n", sub.Endpoint, err)
				}
			} else {
				l.E.Printf("Push notification failed for %v: %v\n", sub.Endpoint, err)
			}
		}
	}
	return nil
}
