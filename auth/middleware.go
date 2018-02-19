package auth

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

var ErrMalformedCred = echo.NewHTTPError(http.StatusBadRequest, "malformed credential")

type Config struct {
	Validator   func(string, string) error
	LookupToken func(string) error
}

func Middleware(cfg Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		const userKey = "user"
		return func(c echo.Context) error {
			// cookie
			sess, _ := session.Get("session", c)
			if !sess.IsNew {
				exp := sess.Values["expireAt"].(int64)
				if !time.Unix(exp, 0).After(time.Now()) {
					return echo.NewHTTPError(http.StatusUnauthorized, "expired session")
				}
				c.Set(userKey, sess.Values["name"])
				return next(c)
			}

			// http header
			auth := c.Request().Header.Get(echo.HeaderAuthorization)
			parts := strings.Fields(auth)
			if len(parts) != 2 {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid authorization header")
			}
			switch strings.ToLower(parts[0]) {
			case "basic":
				b, err := base64.StdEncoding.DecodeString(parts[1])
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "unknown encoding")
				}
				cred := string(b)
				l := strings.Split(cred, ":")
				if len(l) != 2 {
					return ErrMalformedCred
				}
				if err = cfg.Validator(l[0], l[1]); err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, "incorrect username or password")
				}
				c.Set(userKey, l[0])
				return next(c)
			case "bearer":
				err := cfg.LookupToken(parts[1])
				if err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, "expired session: please login again")
				}
				c.Set(userKey, parts[1])
				return next(c)
			default:
				return echo.NewHTTPError(http.StatusBadRequest, "unknown scheme")
			}
		}
	}
}
