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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	oidc "github.com/coreos/go-oidc"
	"github.com/gofiber/fiber/v3"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/milochristiansen/sessionlogger"
	"github.com/teris-io/shortid"
	"golang.org/x/oauth2"
)

// These are all defined in a separate keys.go file that is not checked in to git.
// var ClientID = ""
// var ClientSecret = ""
// var SessionSigningKey = ""
// var SessionEncryptionKey = ""

type loggerKey struct{}
type userKey struct{}

func isTestMode() bool {
	return os.Getenv("RSN_TEST_MODE") == "1"
}

// Auth contains all data and state about the OIDC provider.
type Auth struct {
	Provider *oidc.Provider
	Config   oauth2.Config
	Context  context.Context
}

var AuthData = &Auth{}

func init() {
	AuthData.Context = context.Background()

	if isTestMode() {
		AuthData.Config = oauth2.Config{
			ClientID:     ClientID,
			ClientSecret: ClientSecret,
			RedirectURL:  "http://httpscolonslashslashwww.com/auth/redirect/google",
			Scopes:       []string{oidc.ScopeOpenID, "email"},
		}
		return
	}

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
}

// WhoAmIData holds the data that will be returned by the whoami endpoint.
type WhoAmIData struct {
	Email          string `json:"email"`
	Subject        string `json:"sub"`
	UID            string `json:"uid"`
	PushSubscribed bool   `json:"pushSubscribed"`
}

// AuthMiddleware validates the JWT session cookie, refreshes the OAuth token,
// and stores user info on the context.
func AuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)

		// Read JWT from cookie
		jwtToken := c.Cookies(sessionCookieName)
		if jwtToken == "" {
			l.W.Printf("No session cookie found.\n")
			return c.SendStatus(fiber.StatusForbidden)
		}

		// Decrypt and verify the JWT
		decrypted, err := jwe.Decrypt([]byte(jwtToken), jwe.WithKey(jwa.A256KW, jwtEncKey))
		if err != nil {
			l.W.Printf("Failed to decrypt session: %v\n", err)
			return c.SendStatus(fiber.StatusForbidden)
		}

		token, err := jwt.Parse(decrypted, jwt.WithKey(jwa.HS256, jwtSignKey), jwt.WithValidate(true))
		if err != nil {
			l.W.Printf("Failed to verify session token: %v\n", err)
			return c.SendStatus(fiber.StatusForbidden)
		}

		// Extract session claims
		claims := &SessionClaims{
			Token:        getStringClaim(token, "token"),
			UID:          getStringClaim(token, jwt.SubjectKey),
			Email:        getStringClaim(token, "email"),
			GoogleSub:    getStringClaim(token, "google_sub"),
			PushEndpoint: getStringClaim(token, "push_endpoint"),
		}

		if claims.Token == "" || claims.UID == "" {
			l.W.Printf("Invalid session claims.\n")
			return c.SendStatus(fiber.StatusForbidden)
		}

		// Make sure if the user thinks they are subscribed that they actually are.
		if claims.PushEndpoint != "" && !PushSubscriptionExists(claims.UID, claims.PushEndpoint) {
			claims.PushEndpoint = ""
		}

		// If we are in test mode just barf out what we have and do not call Google APIs.
		if isTestMode() {
			claims.Expiration = time.Now().Add(sessionExpiry)
			claims.IssuedAt = time.Now()

			if err := setSessionCookie(c, claims); err != nil {
				l.W.Printf("Error updating session: %v\n", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			c.Locals(userKey{}, claims)

			return c.Next()
		}

		// Restore OAuth2 token
		var oauthToken oauth2.Token
		if err := json.Unmarshal([]byte(claims.Token), &oauthToken); err != nil {
			l.W.Printf("Failed to parse stored token: %v\n", err)
			return c.SendStatus(fiber.StatusForbidden)
		}

		// Refresh the OAuth2 token (this also validates the refresh token is still good)
		refreshedToken, err := AuthData.Config.TokenSource(AuthData.Context, &oauthToken).Token()
		if err != nil {
			l.W.Printf("Error refreshing OAuth token: %v\n", err)
			return c.SendStatus(fiber.StatusForbidden)
		}

		// Fetch user info to verify the token is valid
		user, err := AuthData.Provider.UserInfo(AuthData.Context, oauth2.StaticTokenSource(refreshedToken))
		if err != nil {
			l.W.Printf("Error fetching user info: %v\n", err)
			return c.SendStatus(fiber.StatusForbidden)
		}

		// Update session with refreshed token
		refreshedTokenJSON, err := json.Marshal(refreshedToken)
		if err != nil {
			l.W.Printf("Error marshaling refreshed token: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		claims.Token = string(refreshedTokenJSON)
		claims.Email = user.Email
		claims.Expiration = time.Now().Add(sessionExpiry)
		claims.IssuedAt = time.Now()

		if err := setSessionCookie(c, claims); err != nil {
			l.W.Printf("Error updating session: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// Store user info on context
		c.Locals(userKey{}, claims)

		return c.Next()
	}
}


// Endpoint functions:
// =====================================================================================================================

// GoogleLoginEndpoint is the endpoint to begin a Google OAuth login.
func GoogleLoginEndpoint(c fiber.Ctx) error {
	l := c.Locals(loggerKey{}).(*sessionlogger.Logger)

	// If we are in test mode and someone tries to login, send them to the mock endpoint instead.
	if isTestMode() {
		returnUrl := c.Query("r", "/")
		mockURL := fmt.Sprintf("/auth/login/mock?email=test@test.com&google_sub=mock-google-sub&r=%s", url.QueryEscape(returnUrl))
		return c.Redirect().To(mockURL)
	}

	state, err := setLoginCookie(c, c.Query("r"))
	if err != nil {
		l.W.Printf("Error creating login state: %v\n", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Redirect().To(AuthData.Config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent")))
}

// LogoutEndpoint will revoke the google OAuth token and delete the user's session cookie.
func LogoutEndpoint(c fiber.Ctx) error {
	l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
	claims := c.Locals(userKey{}).(*SessionClaims)

	returnUrl := c.Query("r")

	// Revoke Google OAuth token.
	if !isTestMode() {
		var oauthToken oauth2.Token
		err := json.Unmarshal([]byte(claims.Token), &oauthToken)
		if err != nil {
			l.W.Printf("Error parsing OAuth token: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if oauthToken.AccessToken != "" {
			vals := url.Values{
				"token": []string{oauthToken.AccessToken},
			}
			resp, err := http.DefaultClient.Do(&http.Request{
				Method: "POST",
				Header: map[string][]string{
					"Content-Type": {"application/x-www-form-urlencoded"},
				},
				Body: io.NopCloser(bytes.NewBufferString(vals.Encode())),
				URL: &url.URL{
					Scheme: "https",
					Host:   "oauth2.googleapis.com",
					Path:   "/revoke",
				},
			})
			if err != nil {
				l.W.Printf("Error revoking OAuth token: %v\n", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			resp.Body.Close()
		}
	}

	// Remove push subscription if one exists.
	if claims.UID != "" && claims.PushEndpoint != "" {
		RemovePushSubscription(claims.UID, claims.PushEndpoint)
	}

	// Clear the session cookie
	c.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	if returnUrl == "" {
		returnUrl = "/"
	}
	return c.Redirect().To(returnUrl)
}

// MockLoginEndpoint creates a fake session cookie that will simulate being logged in when in testing mode.
func MockLoginEndpoint(c fiber.Ctx) error {
	l := c.Locals(loggerKey{}).(*sessionlogger.Logger)

	email := c.Query("email", "test@test.com")
	googleSub := c.Query("google_sub", "mock-google-sub")

	uid := <-userIDService
	l.I.Printf("Mock login: creating session for user %v (email: %v, google_sub: %v)\n", uid, email, googleSub)

	oauthToken := &oauth2.Token{
		AccessToken: "mock-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	tokenJSON, err := json.Marshal(oauthToken)
	if err != nil {
		l.W.Printf("Error marshaling mock token: %v\n", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	returnUrl := c.Query("r", "/")

	claims := &SessionClaims{
		Token:      string(tokenJSON),
		UID:        uid,
		Email:      email,
		GoogleSub:  googleSub,
		Expiration: time.Now().Add(sessionExpiry),
		IssuedAt:   time.Now(),
	}

	if err := setSessionCookie(c, claims); err != nil {
		l.W.Printf("Error creating mock session: %v\n", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Redirect().To(returnUrl)
}

// User ID generator.
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

// GoogleRedirectEndpoint handles the after login redirect from Google OAuth.
func GoogleRedirectEndpoint(c fiber.Ctx) error {
	l := c.Locals(loggerKey{}).(*sessionlogger.Logger)

	// Validate the state value
	state, returnUrl, err := getLoginStateFromCookie(c)
	if err != nil {
		l.W.Printf("Error loading state from login cookie: %v\n", err)
		return c.SendStatus(fiber.StatusForbidden)
	}

	if c.Query("state") != state {
		l.W.Printf("State mismatch. Expected %v Got: %v\n", state, c.Query("state"))
		return c.Status(fiber.StatusBadRequest).SendString("State mismatch.")
	}

	// Convert the code we are given into a token
	token, err := AuthData.Config.Exchange(AuthData.Context, c.Query("code"))
	if err != nil {
		l.W.Printf("No token found: %v\n", err)
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	// Grab the ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		l.W.Println("No id_token field in oauth2 token")
		return c.Status(fiber.StatusInternalServerError).SendString("No id_token field in oauth2 token.")
	}

	oidcConfig := &oidc.Config{
		ClientID: ClientID,
	}

	// and verify it.
	idToken, err := AuthData.Provider.Verifier(oidcConfig).Verify(AuthData.Context, rawIDToken)
	if err != nil {
		l.W.Printf("Failed to verify ID Token %v\n", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to verify ID Token.")
	}

	// Then grab the information we need from it.
	d := &struct {
		Email   string `json:"email"`
		Subject string `json:"sub"`
	}{}
	err = idToken.Claims(d)
	if err != nil {
		l.W.Printf("Failed to read claims: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read claims.")
	}

	// Get user ID from the DB or create a new DB user if one doesn't exist.
	uid := ""
	err = Queries["GetUID"].Preped.QueryRow(UserProviderGoogle, d.Subject).Scan(&uid)
	if err != nil {
		l.I.Printf("Could not find Google user for Subject (%v) in DB, error: %v\n", d.Subject, err)
		uid = <-userIDService
		l.I.Printf("Creating new user with UID: %v\n", uid)

		_, err = Queries["CreatNewUser"].Preped.Exec(uid, UserProviderGoogle, d.Subject)
		if err != nil {
			l.E.Printf("Cannot insert user %v into db, error: %v\n", uid, err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create user.")
		}
	}

	// Store the token in the session for later.
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		l.W.Printf("Error marshaling token: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to marshal token.")
	}

	claims := &SessionClaims{
		Token:      string(tokenJSON),
		UID:        uid,
		Email:      d.Email,
		GoogleSub:  d.Subject,
		Expiration: time.Now().Add(sessionExpiry),
		IssuedAt:   time.Now(),
	}

	if err := setSessionCookie(c, claims); err != nil {
		l.W.Printf("Error creating session: %v\n", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// Clear the login cookie
	c.Cookie(&fiber.Cookie{
		Name:     loginCookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	if returnUrl == "" {
		returnUrl = "/"
	}
	return c.Redirect().To(returnUrl)
}
