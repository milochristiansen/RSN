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
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/milochristiansen/sessionlogger"
)

const MaxBodyBytes = int64(65536)

// LoggerMiddleware creates a session logger and stores it on the context.
func LoggerMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		l := sessionlogger.NewSessionLogger(c.Path())
		c.Locals(loggerKey{}, l)
		return c.Next()
	}
}

func main() {
	app := fiber.New()

	// Auth routes (logger only, no auth middleware)
	authGroup := app.Group("/auth", LoggerMiddleware())
	authGroup.Get("/login/google", GoogleLoginEndpoint)
	authGroup.Get("/redirect/google", GoogleRedirectEndpoint)
	authGroup.Get("/logout", LogoutEndpoint)

	// Special fake login URL for testing. In testing mode any user that hits a login url is redirected here.
	if isTestMode() {
		authGroup.Get("/login/mock", MockLoginEndpoint)
	}

	// /auth/whoami is special, as it is an authenticated route in an otherwise unauthenticated group.
	authGroup.Get("/whoami", AuthMiddleware(), func(c fiber.Ctx) error {
		claims := c.Locals(userKey{}).(*SessionClaims)

		jd, err := json.MarshalIndent(&WhoAmIData{
			Email:          claims.Email,
			Subject:        claims.GoogleSub,
			UID:            claims.UID,
			PushSubscribed: claims.PushEndpoint != "",
		}, "", "    ")
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.Send(jd)
	})

	// Protected API routes (logger + auth)
	apiGroup := app.Group("/api", LoggerMiddleware(), AuthMiddleware())

	// GET /api/feeds - list all feeds
	apiGroup.Get("/feeds", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		feeds := FeedList(l, user.UID)
		if feeds == nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(feeds)
	})

	// GET /api/feeds/:id - get feed details
	apiGroup.Get("/feeds/:id", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		id := c.Params("id")
		if id == "" {
			l.W.Printf("Missing feed ID.\n")
			return c.SendStatus(fiber.StatusBadRequest)
		}

		details := FeedDetails(l, user.UID, id)
		if details == nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(details)
	})

	// GET /api/feeds/:id/articles - get articles for a feed
	apiGroup.Get("/feeds/:id/articles", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		id := c.Params("id")
		if id == "" {
			l.W.Printf("Missing feed ID.\n")
			return c.SendStatus(fiber.StatusBadRequest)
		}

		articles := FeedArticles(l, user.UID, id)
		if articles == nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(articles)
	})

	// POST /api/feeds - subscribe to a feed
	apiGroup.Post("/feeds", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		if int64(len(c.Body())) > MaxBodyBytes {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		data := &FeedSubscribeData{}
		err := c.Bind().Body(data)
		if err != nil {
			l.W.Printf("Error parsing feed subscribe body. Error: %v\n", err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if data.Name == "" {
			l.W.Printf("No feed name given.\n")
			return c.SendStatus(fiber.StatusBadRequest)
		}

		c.Status(FeedSubscribe(l, user.UID, data.URL, data.Name))
		return nil
	})

	// DELETE /api/feeds/:id - unsubscribe from a feed
	apiGroup.Delete("/feeds/:id", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		id := c.Params("id")
		if id == "" {
			l.W.Printf("Missing feed ID.\n")
			return c.SendStatus(fiber.StatusBadRequest)
		}

		c.Status(FeedUnsub(l, user.UID, id))
		return nil
	})

	// PATCH /api/feeds/:id - pause/unpause/rename a feed
	apiGroup.Patch("/feeds/:id", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		id := c.Params("id")
		if id == "" {
			l.W.Printf("Missing feed ID.\n")
			return c.SendStatus(fiber.StatusBadRequest)
		}

		body := &FeedPatchData{}
		err := c.Bind().Body(body)
		if err != nil {
			l.W.Printf("Error parsing feed patch body. Error: %v\n", err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if body.Name != "" {
			c.Status(FeedRename(l, user.UID, id, body.Name))
			return nil
		}
		if body.Paused != nil {
			if *body.Paused {
				c.Status(FeedPause(l, user.UID, id))
				return nil
			}
			c.Status(FeedUnpause(l, user.UID, id))
			return nil
		}

		return c.SendStatus(fiber.StatusBadRequest)
	})

	// PATCH /api/articles/:id - mark article as read/unread
	apiGroup.Patch("/articles/:id", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		id := c.Params("id")
		if id == "" {
			l.W.Printf("Missing article ID.\n")
			return c.SendStatus(fiber.StatusBadRequest)
		}

		body := &ArticlePatchData{}
		err := c.Bind().Body(body)
		if err != nil {
			l.W.Printf("Error parsing article patch body. Error: %v\n", err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if body.Read != nil {
			if *body.Read {
				c.Status(ArticleMarkRead(l, user.UID, id))
				return nil
			}
			c.Status(ArticleMarkUnread(l, user.UID, id))
			return nil
		}

		return c.SendStatus(fiber.StatusBadRequest)
	})

	// GET /api/getunread
	apiGroup.Get("/getunread", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		articles := GetUnread(l, user.UID)
		if articles == nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(articles)
	})

	// GET /api/recentread
	apiGroup.Get("/recentread", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		user := c.Locals(userKey{}).(*SessionClaims)

		articles := GetRecentRead(l, user.UID)
		if articles == nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(articles)
	})

	// POST /api/push/subscription
	apiGroup.Post("/push/subscription", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		claims := c.Locals(userKey{}).(*SessionClaims)

		var data PushSubscribeRequest
		if err := c.Bind().Body(&data); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if data.Endpoint == "" || data.P256DH == "" || data.Auth == "" {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if err := RegisterPushSubscription(claims.UID, data.Endpoint, data.P256DH, data.Auth); err != nil {
			l.E.Printf("Failed to register push subscription for user %v: %v\n", claims.UID, err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if err := UpdateSessionPushEndpoint(c, data.Endpoint); err != nil {
			l.E.Printf("Failed to update session push endpoint for user %v: %v\n", claims.UID, err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.SendStatus(fiber.StatusOK)
	})

	// GET /api/push/vapid
	apiGroup.Get("/push/vapid", func(c fiber.Ctx) error {
		return c.JSON(map[string]string{
			"publicKey": VAPIDPublicKey,
		})
	})

	// DELETE /api/push/subscription
	apiGroup.Delete("/push/subscription", func(c fiber.Ctx) error {
		l := c.Locals(loggerKey{}).(*sessionlogger.Logger)
		claims := c.Locals(userKey{}).(*SessionClaims)

		if claims.PushEndpoint == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}

		if err := RemovePushSubscription(claims.UID, claims.PushEndpoint); err != nil {
			l.E.Printf("Failed to remove push subscription for user %v: %v\n", claims.UID, err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if err := UpdateSessionPushEndpoint(c, ""); err != nil {
			l.E.Printf("Failed to clear session push endpoint for user %v: %v\n", claims.UID, err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.SendStatus(fiber.StatusOK)
	})

	if isTestMode() {
		siteDir := "../Site"

		app.Use("/", func(c fiber.Ctx) error {
			path := c.Path()

			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/auth") {
				return c.Next()
			}

			filePath := filepath.Join(siteDir, path)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				return c.SendFile(filePath)
			}

			return c.Type("html").SendFile(filepath.Join(siteDir, "index.html"))
		})
	}

	go Background()

	port := ":80"
	if isTestMode() {
		port = ":8080"
	}
	if err := app.Listen(port); err != nil {
		panic(err)
	}
}
