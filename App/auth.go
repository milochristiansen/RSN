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
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"

	oidc "github.com/coreos/go-oidc"
	"github.com/teris-io/shortid"

	"github.com/gorilla/sessions"

	"github.com/milochristiansen/sessionlogger"
)

// These are all defined in a separate keys.go file that is not checked in to git.
// var ClientID = ""
// var ClientSecret = ""
// var SessionCookieKeyA = ""
// var SessionCookieKeyB = ""

// Auth contains all data and state about the OIDC provider.
type Auth struct {
	Provider *oidc.Provider
	Config   oauth2.Config
	Context  context.Context
}

var AuthData = &Auth{}

var SessionStore sessions.Store

func init() {
	AuthData.Context = context.Background()

	provider, err := oidc.NewProvider(AuthData.Context, "https://accounts.google.com")
	if err != nil {
		panic(err)
	}
	AuthData.Provider = provider

	AuthData.Config = oauth2.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  "http://httpscolonslashslashwww.com/auth/redirect/google",
		Scopes:       []string{oidc.ScopeOpenID, "email"},
	}

	rawkey := []byte(SessionCookieKeyA)
	keyA := make([]byte, hex.DecodedLen(len(rawkey)))
	_, err = hex.Decode(keyA, rawkey)
	if err != nil {
		panic("Could not load session key A.\n" + err.Error())
	}
	rawkey = []byte(SessionCookieKeyB)
	keyB := make([]byte, hex.DecodedLen(len(rawkey)))
	_, err = hex.Decode(keyB, rawkey)
	if err != nil {
		panic("Could not load session key B.\n" + err.Error())
	}

	SessionStore = sessions.NewCookieStore(keyA, keyB)
}

// SetSessionData creates a user session containing a token and user DB ID.
func SetSessionData(l *sessionlogger.Logger, w http.ResponseWriter, r *http.Request, token *oauth2.Token, uid string) int {
	session, _ := SessionStore.Get(r, "rsn-session")

	tokenB, err := json.Marshal(token)
	if err != nil {
		l.W.Printf("Error marshaling token: %v\n", err)
		return http.StatusInternalServerError
	}
	session.Values["token"] = string(tokenB)

	session.Values["uid"] = uid

	err = session.Save(r, w)
	if err != nil {
		l.W.Printf("Error saving session, error: %v\n", err)
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

// WhoAmIData holds the data that will be returned by the whoami endpoint.
type WhoAmIData struct {
	// Fetched fresh every time the whoami endpoint is called
	Email   string // User's email
	Subject string // The remote provider user ID

	// Stored in the session
	UID string // User DB ID
}

// GetSessionData returns either data about the current user and StatusOK, or nil and StatusForbidden or StatusInternalServerError
func GetSessionData(l *sessionlogger.Logger, w http.ResponseWriter, r *http.Request) (*WhoAmIData, int) {
	session, _ := SessionStore.Get(r, "rsn-session")

	// Fields
	// UID: User ID (for DB)
	// Token: JSON encoded token

	// Get token from session
	tokenR, ok := session.Values["token"].(string)
	if !ok || tokenR == "" {
		l.W.Printf("Could not load token from session.\n")
		return nil, http.StatusForbidden
	}

	// Parse the token
	token := &oauth2.Token{}
	err := json.Unmarshal([]byte(tokenR), token)
	if err != nil {
		l.W.Printf("Error parsing token: %v\n", err)
		return nil, http.StatusInternalServerError
	}

	// Refresh the token if needed, and use it to get the user info.
	source := AuthData.Config.TokenSource(AuthData.Context, token)
	user, err := AuthData.Provider.UserInfo(AuthData.Context, source)
	if err != nil {
		l.W.Printf("Error fetching user info: %v\n", err)
		return nil, http.StatusInternalServerError
	}

	token, err = source.Token()
	if err != nil {
		l.W.Printf("Error getting current token for restorage: %v\n", err)
		return nil, http.StatusInternalServerError
	}

	// Save the possibly refreshed token
	tokenB, err := json.Marshal(token)
	if err != nil {
		l.W.Printf("Error marshaling token: %v\n", err)
		return nil, http.StatusInternalServerError
	}
	session.Values["token"] = string(tokenB)

	// Get the UID from the session
	uid, ok := session.Values["uid"].(string)
	if !ok || uid == "" {
		l.W.Println("Could not load UID from session.")
		return nil, http.StatusInternalServerError // If we got this far it would be very odd not to have a valid UID
	}

	// Save Session
	err = session.Save(r, w)
	if err != nil {
		l.W.Printf("Error saving session for user %v, error: %v\n", user, err)
		return nil, http.StatusInternalServerError
	}

	// Return whoami data from fetched user info
	return &WhoAmIData{
		Email:   user.Email,
		Subject: user.Subject,
		UID:     uid,
	}, http.StatusOK
}

// DeleteSessionData deletes the user session and revokes the token it contains.
func DeleteSessionData(l *sessionlogger.Logger, w http.ResponseWriter, r *http.Request) int {
	session, _ := SessionStore.Get(r, "rsn-session")

	// Get token from session
	tokenR, ok := session.Values["token"].(string)
	if !ok || tokenR == "" {
		l.W.Printf("Could not load token from session.\n")
		return http.StatusBadRequest // Can't logout if you aren't logged in.
	}

	// Parse the token
	token := &oauth2.Token{}
	err := json.Unmarshal([]byte(tokenR), token)
	if err != nil {
		l.W.Printf("Error parsing token: %v\n", err)
		// Not a bad request since this is stored in an encrypted cookie. If it is mangled it
		// pretty much has to be the server's fault.
		return http.StatusInternalServerError
	}

	// Revoke token
	// I could not find a generic revocation API in either the oauth or oidc libraries, so do it the caveman way.
	_, err = http.DefaultClient.Do(&http.Request{
		Method: "GET",
		Header: map[string][]string{
			"Content-type": {"application/x-www-form-urlencoded"},
		},
		URL: &url.URL{
			Scheme:   "https",
			Host:     "oauth2.googleapis.com",
			Path:     "/revoke",
			RawQuery: "token=" + token.AccessToken,
		},
	})
	if err != nil {
		l.W.Printf("Error revoking token: %v\n", err)
		return http.StatusInternalServerError
	}

	// Delete session cookie
	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		l.W.Printf("Error deleting login session, error: %v\n", err)
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

// GetLoginState returns the login state value and return url for the current user and 200, or empty strings and an HTTP error code.
func GetLoginState(l *sessionlogger.Logger, w http.ResponseWriter, r *http.Request) (string, string, int) {
	session, _ := SessionStore.Get(r, "rsn-login")

	state, ok := session.Values["state"].(string)
	if !ok || state == "" {
		l.W.Printf("Error loading state from login session.\n")
		return "", "", http.StatusForbidden
	}

	// Optional.
	returnUrl, ok := session.Values["return"].(string)

	// Delete the temporary login session
	session.Options.MaxAge = -1

	err := session.Save(r, w)
	if err != nil {
		l.W.Printf("Error saving login session, error: %v\n", err)
		return "", "", http.StatusInternalServerError
	}
	return state, returnUrl, http.StatusOK
}

// SetLoginState returns a new login state value for the current user and 200, or an empty string and a HTTP error code.
func SetLoginState(l *sessionlogger.Logger, w http.ResponseWriter, r *http.Request, returnUrl string) (string, int) {
	session, _ := SessionStore.Get(r, "rsn-login")

	// Encode eight random bytes as hex and use that for the state value.
	stateRaw := make([]byte, 8)
	rand.Read(stateRaw)
	stateEnc := make([]byte, 16)
	hex.Encode(stateEnc, stateRaw)
	state := string(stateEnc)

	session.Values["state"] = state
	session.Values["return"] = returnUrl

	err := session.Save(r, w)
	if err != nil {
		l.W.Printf("Error saving login session, error: %v\n", err)
		return "", http.StatusInternalServerError
	}
	return state, http.StatusOK
}

// Endpoint functions:
// =====================================================================================================================

func GoogleLoginEndpoint(w http.ResponseWriter, r *http.Request) {
	l := sessionlogger.NewSessionLogger("/api/login/google")

	// Generate a random state value, store it in a temporary session along with the return URL, and
	// then get a copy of the state to give to the provider.
	state, status := SetLoginState(l, w, r, r.URL.Query().Get("r"))
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}

	// Redirect to the provider with our state value and a request for offline access (aka, give me a refresh token)
	http.Redirect(w, r, AuthData.Config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce), http.StatusFound)
}

func LogoutEndpoint(w http.ResponseWriter, r *http.Request) {
	l := sessionlogger.NewSessionLogger("/api/login/logout")

	returnUrl := r.URL.Query().Get("r")

	// Delete the session data and revoke the token contained within.
	status := DeleteSessionData(l, w, r)
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}

	// Redirect to the return URL.
	if returnUrl == "" {
		returnUrl = "/"
	}
	http.Redirect(w, r, returnUrl, http.StatusFound)
}

var userIDService <-chan string

func init() {
	go func() {
		c := make(chan string)
		userIDService = c

		idsource := shortid.MustNew(9, shortid.DefaultABC, uint64(time.Now().UnixNano()))

		for {
			c <- idsource.MustGenerate()
		}
	}()
}

func GoogleRedirectEndpoint(w http.ResponseWriter, r *http.Request) {
	l := sessionlogger.NewSessionLogger("/api/login/google/redirect")

	// Get the current state and return URL from the temporary session, then delete the session.
	state, returnUrl, status := GetLoginState(l, w, r)
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}

	// Make sure the state from the provider matches the one from the temporary session.
	if r.URL.Query().Get("state") != state {
		l.W.Printf("State mismatch. Expected %v Got: %v\n", state, r.URL.Query().Get("state"))
		http.Error(w, "State mismatch.", http.StatusBadRequest)
		return
	}

	// Turn the code we received into a token.
	token, err := AuthData.Config.Exchange(AuthData.Context, r.URL.Query().Get("code"))
	if err != nil {
		l.W.Printf("No token found: %v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Verify the token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		l.W.Println("No id_token field in oauth2 token")
		http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
		return
	}

	oidcConfig := &oidc.Config{
		ClientID: ClientID,
	}

	idToken, err := AuthData.Provider.Verifier(oidcConfig).Verify(AuthData.Context, rawIDToken)
	if err != nil {
		l.W.Printf("Failed to verify ID Token %v\n", err)
		http.Error(w, "Failed to verify ID Token.", http.StatusInternalServerError)
		return
	}

	// Grab the email and provider user ID from the claims.
	d := &struct {
		Email   string `json:"email"`
		Subject string `json:"sub"`
	}{}
	err = idToken.Claims(d)
	if err != nil {
		l.W.Printf("Failed to read claims: %v\n", err)
		http.Error(w, "Failed to read claims.", http.StatusInternalServerError)
		return
	}

	// Look up user ID in database, or create new user based on the provider information.
	uid := ""
	err = Queries["GetUID"].Preped.QueryRow(UserProviderGoogle, d.Subject).Scan(&uid)
	if err != nil {
		l.I.Printf("Could not find Google user for Subject (%v) in DB, error: %v\n", d.Subject, err)
		uid = <-userIDService
		l.I.Printf("Creating new user with UID: %v\n", uid)

		_, err = Queries["CreatNewUser"].Preped.Exec(uid, UserProviderGoogle, d.Subject)
		if err != nil {
			l.E.Printf("Cannot insert user %v into db, error: %v\n", uid, err)
			http.Error(w, "Failed to create user.", http.StatusInternalServerError)
			return
		}
	}

	// Now, initialize the user session so whoami functions.
	status = SetSessionData(l, w, r, token, uid)
	if status != http.StatusOK {
		http.Error(w, "Session save failed.", status)
		return
	}

	// Redirect the user to the return URL.
	if returnUrl == "" {
		returnUrl = "/"
	}
	http.Redirect(w, r, returnUrl, http.StatusFound)
}

func WhoAmIEndpoint(w http.ResponseWriter, r *http.Request) {
	l := sessionlogger.NewSessionLogger("/api/whoami")

	// Get the token from the session, then use it to get user info from the provider.
	data, status := GetSessionData(l, w, r)
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}

	// And.. That's kinda it. This endpoint does a lot of stuff, but it is all boring admin work.
	jd, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(jd)
}
