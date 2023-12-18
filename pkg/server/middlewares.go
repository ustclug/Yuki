package server

import (
	"log/slog"

	"github.com/labstack/echo/v4"
)

const ctxKeyLogger = "yukid-logger"

func setLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqID := c.Response().Header().Get(echo.HeaderXRequestID)
			c.Set(
				ctxKeyLogger,
				logger.With(
					slog.String("req_id", reqID),
				))
			return next(c)
		}
	}
}

func getLogger(c echo.Context) *slog.Logger {
	return c.Get(ctxKeyLogger).(*slog.Logger)
}
