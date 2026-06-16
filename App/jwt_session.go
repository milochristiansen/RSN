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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Cookie name constants:
const sessionCookieName = "rsn-session"
const loginCookieName = "rsn-login"

// Expiration duration constants:
const loginExpiry = 5 * time.Minute
const sessionExpiry = 7 * 24 * time.Hour

// SessionClaims holds the decrypted session data stored inside the JWT.
type SessionClaims struct {
	Token        string    `json:"token"`
	UID          string    `json:"uid"`
	Email        string    `json:"email"`
	GoogleSub    string    `json:"google_sub"`
	PushEndpoint string    `json:"push_endpoint"`
	Expiration   time.Time `json:"exp"`
	IssuedAt     time.Time `json:"iat"`
}

// LoginClaims holds the OAuth CSRF state and return URL stored inside the login JWT.
type LoginClaims struct {
	State      string    `json:"state"`
	ReturnURL  string    `json:"return_url"`
	Expiration time.Time `json:"exp"`
	IssuedAt   time.Time `json:"iat"`
}

// Decoded hex keys for JWT signing and encryption:
var jwtSignKey []byte
var jwtEncKey []byte

func init() {
	var err error
	jwtSignKey, err = hex.DecodeString(SessionSigningKey)
	if err != nil {
		panic("Invalid SessionSigningKey: " + err.Error())
	}
	jwtEncKey, err = hex.DecodeString(SessionEncryptionKey)
	if err != nil {
		panic("Invalid SessionEncryptionKey: " + err.Error())
	}
}

// getStringClaim extracts a string claim from a JWT token.
func getStringClaim(t jwt.Token, key string) string {
	v, _ := t.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// setSessionCookie creates a new JWT and sets it as the session cookie.
func setSessionCookie(c fiber.Ctx, claims *SessionClaims) error {
	t := jwt.New()
	t.Set(jwt.SubjectKey, claims.UID)
	t.Set("token", claims.Token)
	t.Set("email", claims.Email)
	t.Set("google_sub", claims.GoogleSub)
	t.Set("push_endpoint", claims.PushEndpoint)
	t.Set(jwt.ExpirationKey, claims.Expiration)
	t.Set(jwt.IssuedAtKey, claims.IssuedAt)

	signed, err := jwt.Sign(t, jwt.WithKey(jwa.HS256, jwtSignKey))
	if err != nil {
		return err
	}

	encrypted, err := jwe.Encrypt(signed, jwe.WithKey(jwa.A256KW, jwtEncKey), jwe.WithContentEncryption(jwa.A256GCM))
	if err != nil {
		return err
	}

	// Set the JWT as the session cookie
	c.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    string(encrypted),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
		Expires:  claims.Expiration,
	})

	return nil
}

// UpdateSessionPushEndpoint updates the push endpoint in the session cookie.
func UpdateSessionPushEndpoint(c fiber.Ctx, endpoint string) error {
	claims := c.Locals(userKey{}).(*SessionClaims)
	if claims == nil {
		return fmt.Errorf("no session claims found")
	}

	claims.PushEndpoint = endpoint

	return setSessionCookie(c, claims)
}

// generateLoginState generates a random 16-character hex state string.
func generateLoginState() string {
	stateRaw := make([]byte, 8)
	rand.Read(stateRaw)
	stateEnc := make([]byte, 16)
	hex.Encode(stateEnc, stateRaw)
	return string(stateEnc)
}

// setLoginCookie creates a login state JWT and sets it as the login cookie.
func setLoginCookie(c fiber.Ctx, returnUrl string) (string, error) {
	state := generateLoginState()

	t := jwt.New()
	t.Set("state", state)
	t.Set("return_url", returnUrl)
	t.Set(jwt.ExpirationKey, time.Now().Add(loginExpiry))
	t.Set(jwt.IssuedAtKey, time.Now())

	signed, err := jwt.Sign(t, jwt.WithKey(jwa.HS256, jwtSignKey))
	if err != nil {
		return "", err
	}

	encrypted, err := jwe.Encrypt(signed, jwe.WithKey(jwa.A256KW, jwtEncKey), jwe.WithContentEncryption(jwa.A256GCM))
	if err != nil {
		return "", err
	}

	c.Cookie(&fiber.Cookie{
		Name:     loginCookieName,
		Value:    string(encrypted),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax", // This needs to be Lax so the redirect URL handler can see the cookie.
		Path:     "/",
		Expires:  time.Now().Add(loginExpiry),
	})

	return state, nil
}

// getLoginStateFromCookie reads and validates the login state JWT, returning the state and return URL.
// It also clears the login cookie.
func getLoginStateFromCookie(c fiber.Ctx) (string, string, error) {
	jwtToken := c.Cookies(loginCookieName)
	if jwtToken == "" {
		return "", "", fmt.Errorf("no login cookie found")
	}

	decrypted, err := jwe.Decrypt([]byte(jwtToken), jwe.WithKey(jwa.A256KW, jwtEncKey))
	if err != nil {
		return "", "", err
	}

	token, err := jwt.Parse(decrypted, jwt.WithKey(jwa.HS256, jwtSignKey), jwt.WithValidate(true))
	if err != nil {
		return "", "", err
	}

	state := getStringClaim(token, "state")
	returnURL := getStringClaim(token, "return_url")

	if state == "" {
		return "", "", fmt.Errorf("no state in login cookie")
	}

	// Clear the login cookie
	c.Cookie(&fiber.Cookie{
		Name:     loginCookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	return state, returnURL, nil
}
