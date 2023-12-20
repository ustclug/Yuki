package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type echoValidator func(i any) error

func (v echoValidator) Validate(i any) error {
	if err := v(i); err != nil {
		// Optionally, you could return the error to give each route more control over the status code
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}
