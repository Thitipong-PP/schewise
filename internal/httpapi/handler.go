// Package httpapi implements the public Fiber HTTP boundary.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const CookieName = "__Host-schewise_session"

type Config struct {
	Auth   *auth.Service
	Logger *zap.Logger
	Origin string
}

type server struct{ Config }

func NewApp(config Config) *fiber.App {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	s := &server{Config: config}
	app := fiber.New(fiber.Config{
		AppName: "Schewise", Immutable: true, BodyLimit: 8 * 1024, ReadTimeout: 15 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var fe *fiber.Error
			if errors.As(err, &fe) {
				code = fe.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": http.StatusText(code)})
		},
	})
	app.Use(s.requestLog)
	app.Use(recover.New())
	app.Use(func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		return c.Next()
	})
	health := func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok", "service": "schewise-api"}) }
	app.Get("/api/v1/health", health)
	app.Head("/api/v1/health", health)
	app.All("/api/v1/health", func(c fiber.Ctx) error { c.Set("Allow", "GET, HEAD"); return fiber.ErrMethodNotAllowed })
	app.Use("/api/v1/auth", s.available)
	app.Get("/api/v1/auth/csrf", s.rateLimit(60), s.csrf)
	app.Post("/api/v1/auth/login", s.rateLimit(10), s.mutation, s.login)
	app.Post("/api/v1/auth/register", s.rateLimit(20), s.mutation, s.authenticated, s.permission(rbac.UserManage), s.register)
	app.Post("/api/v1/auth/logout", s.mutation, s.logout)
	app.Get("/api/v1/auth/me", s.authenticated, s.me)
	app.Delete("/api/v1/users/:id/sessions", s.available, s.mutation, s.authenticated, s.permission(rbac.UserManage), s.revokeSessions)
	return app
}

func (s *server) requestLog(c fiber.Ctx) error {
	start := time.Now()
	requestID := uuid.NewString()
	c.Locals("request_id", requestID)
	c.Set("X-Request-ID", requestID)
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()
	c.SetContext(ctx)
	err := c.Next()
	if err != nil {
		if handlerErr := c.App().ErrorHandler(c, err); handlerErr != nil {
			return handlerErr
		}
	}
	// Route patterns exclude query strings, bodies, headers, and cookies.
	// Unknown paths are redacted as they may contain sensitive values.
	path := "unmatched"
	if c.Route() != nil && c.Route().Path != "/" {
		path = c.Route().Path
	}
	s.Logger.Info("http.request", zap.String("request_id", requestID), zap.String("method", c.Method()),
		zap.String("route", path), zap.Int("status", c.Response().StatusCode()), zap.Duration("duration", time.Since(start)),
		zap.String("actor_id", actorID(c)), zap.String("remote_ip", c.IP()))
	return nil
}

func actorID(c fiber.Ctx) string {
	if session, ok := c.Locals("session").(*auth.Session); ok && session.User != nil {
		return session.User.ID
	}
	return ""
}

func event(c fiber.Ctx, action, outcome string) auth.Event {
	id, _ := c.Locals("request_id").(string)
	return auth.Event{RequestID: id, ActorID: actorID(c), Action: action, Outcome: outcome, RemoteIP: c.IP()}
}

func (s *server) available(c fiber.Ctx) error {
	if s.Auth == nil {
		return fiber.ErrServiceUnavailable
	}
	return c.Next()
}

func (s *server) rateLimit(max int) fiber.Handler {
	return limiter.New(limiter.Config{Max: max, Expiration: time.Minute, LimitReached: func(c fiber.Ctx) error {
		if err := s.Auth.Audit(c.Context(), event(c, "auth.rate_limited", "denied")); err != nil {
			return s.failure(c, err)
		}
		return c.Status(429).JSON(fiber.Map{"error": "Too many requests. Try again later."})
	}})
}

func (s *server) session(c fiber.Ctx) (*auth.Session, error) {
	if value, ok := c.Locals("session").(*auth.Session); ok {
		return value, nil
	}
	value, err := s.Auth.Lookup(c.Context(), c.Cookies(CookieName))
	if err == nil {
		c.Locals("session", value)
	}
	return value, err
}

func (s *server) csrf(c fiber.Ctx) error {
	session, err := s.session(c)
	if errors.Is(err, auth.ErrUnauthorized) {
		session, err = s.Auth.Anonymous(c.Context())
		if err == nil {
			setCookie(c, session)
			c.Locals("session", session)
		}
	}
	if err != nil {
		return s.failure(c, err)
	}
	return c.JSON(fiber.Map{"csrf_token": session.CSRF})
}

func setCookie(c fiber.Ctx, session *auth.Session) {
	c.Cookie(&fiber.Cookie{Name: CookieName, Value: session.Token, Path: "/", HTTPOnly: true, Secure: true, SameSite: "Lax", Expires: session.ExpiresAt})
}

func clearCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{Name: CookieName, Value: "", Path: "/", HTTPOnly: true, Secure: true, SameSite: "Lax", MaxAge: -1, Expires: time.Unix(1, 0)})
}

func (s *server) mutation(c fiber.Ctx) error {
	if c.Get("Origin") != s.Origin {
		return s.deny(c, "auth.csrf", "invalid_origin", 403)
	}
	session, err := s.session(c)
	if errors.Is(err, auth.ErrUnauthorized) {
		return s.deny(c, "auth.csrf", "missing_session", 403)
	}
	if err != nil {
		return s.failure(c, err)
	}
	if !auth.VerifyCSRF(session, c.Get("X-CSRF-Token")) {
		return s.deny(c, "auth.csrf", "invalid_token", 403)
	}
	return c.Next()
}

func (s *server) authenticated(c fiber.Ctx) error {
	if s.Auth == nil {
		return fiber.ErrServiceUnavailable
	}
	session, err := s.session(c)
	if errors.Is(err, auth.ErrUnauthorized) || (err == nil && session.User == nil) {
		return s.deny(c, "auth.access", "unauthenticated", 401)
	}
	if err != nil {
		return s.failure(c, err)
	}
	return c.Next()
}

func (s *server) permission(permission rbac.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		session, _ := s.session(c)
		allowed, err := s.Auth.Can(c.Context(), session.User, permission, rbac.RootScope())
		if err != nil {
			return s.failure(c, err)
		}
		if !allowed {
			return s.deny(c, "auth.authorization", "denied", 403)
		}
		return c.Next()
	}
}

func (s *server) deny(c fiber.Ctx, action, outcome string, status int) error {
	if err := s.Auth.Audit(c.Context(), event(c, action, outcome)); err != nil {
		return s.failure(c, err)
	}
	return c.Status(status).JSON(fiber.Map{"error": http.StatusText(status)})
}

func (s *server) failure(c fiber.Ctx, err error) error {
	// Driver errors can contain data or connection strings. Never serialize them.
	code := 503
	message := "Service temporarily unavailable."
	switch {
	case errors.Is(err, auth.ErrCredentials):
		code = 401
		message = "Invalid email or password."
	case errors.Is(err, auth.ErrUnauthorized):
		code = 401
		message = "Authentication required."
	case errors.Is(err, auth.ErrForbidden):
		code = 403
		message = "Required permission missing."
	case errors.Is(err, auth.ErrInvalidInput):
		code = 400
		message = "Provide a valid email and a password of at least 12 characters and at most 256 bytes."
	case errors.Is(err, auth.ErrConflict):
		code = 409
		message = "An account with that email already exists."
	case errors.Is(err, auth.ErrNotFound):
		code = 404
		message = "User not found."
	case errors.Is(err, auth.ErrBusy):
		code = 429
		message = "Authentication is busy. Try again later."
	}
	if code == 503 {
		s.Logger.Error("http.operation_failed", zap.String("request_id", c.Locals("request_id").(string)), zap.Int("status", code))
	}
	return c.Status(code).JSON(fiber.Map{"error": message})
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func readCredentials(c fiber.Ctx) (credentials, error) {
	var input credentials
	media, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return input, auth.ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return input, auth.ErrInvalidInput
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return input, auth.ErrInvalidInput
	}
	if len(input.Email) > 254 || len(input.Password) > 256 {
		return input, auth.ErrInvalidInput
	}
	return input, nil
}

func (s *server) login(c fiber.Ctx) error {
	input, err := readCredentials(c)
	if err != nil {
		return s.deny(c, "auth.login", "invalid_input", 400)
	}
	previous, _ := s.session(c)
	session, err := s.Auth.Login(c.Context(), previous, input.Email, input.Password, event(c, "auth.login", "attempt"))
	if err != nil {
		return s.failure(c, err)
	}
	c.Locals("session", session)
	setCookie(c, session)
	return c.JSON(fiber.Map{"user": session.User})
}

func (s *server) register(c fiber.Ctx) error {
	input, err := readCredentials(c)
	if err != nil {
		return s.deny(c, "user.register", "invalid_input", 400)
	}
	session, _ := s.session(c)
	user, err := s.Auth.Register(c.Context(), session.User, input.Email, input.Password, event(c, "user.register", "attempt"))
	if err != nil {
		if auditErr := s.Auth.Audit(c.Context(), event(c, "user.register", "failure")); auditErr != nil {
			return s.failure(c, auditErr)
		}
		return s.failure(c, err)
	}
	return c.Status(201).JSON(fiber.Map{"user": user})
}

func (s *server) logout(c fiber.Ctx) error {
	session, _ := s.session(c)
	if err := s.Auth.Logout(c.Context(), session, event(c, "auth.logout", "attempt")); err != nil {
		return s.failure(c, err)
	}
	clearCookie(c)
	return c.SendStatus(204)
}

func (s *server) me(c fiber.Ctx) error {
	session, _ := s.session(c)
	return c.JSON(fiber.Map{"user": session.User})
}

func (s *server) revokeSessions(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return s.deny(c, "session.revoke_user", "invalid_input", 400)
	}
	session, _ := s.session(c)
	if err := s.Auth.RevokeUserSessions(c.Context(), session.User, id.String(), event(c, "session.revoke_user", "attempt")); err != nil {
		return s.failure(c, err)
	}
	if id.String() == session.User.ID {
		clearCookie(c)
	}
	return c.SendStatus(204)
}
