package server

import (
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
)

const ctxKeyLogger = "yukid-logger"

func setLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqID := c.Response().Header().Get(echo.HeaderXRequestID)
			attrs := []any{
				slog.String("req_id", reqID),
			}
			routePath := c.Path()
			if len(routePath) > 0 {
				attrs = append(
					attrs,
					slog.String("endpoint", fmt.Sprintf("%s %s", c.Request().Method, routePath)),
				)
			}
			c.Set(
				ctxKeyLogger,
				logger.With(attrs...),
			)
			return next(c)
		}
	}
}

func getLogger(c echo.Context) *slog.Logger {
	return c.Get(ctxKeyLogger).(*slog.Logger)
}
