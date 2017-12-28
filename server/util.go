package server

import (
	"net/http"

	"github.com/labstack/echo"
)

func BadRequest(msg ...interface{}) error {
	return echo.NewHTTPError(http.StatusBadRequest, msg...)
}

func NotFound(msg ...interface{}) error {
	return echo.NewHTTPError(http.StatusNotFound, msg...)
}

func NotAcceptable(msg ...interface{}) error {
	return echo.NewHTTPError(http.StatusNotAcceptable, msg...)
}

func Conflict(msg ...interface{}) error {
	return echo.NewHTTPError(http.StatusConflict, msg...)
}

func Forbidden(msg ...interface{}) error {
	return echo.NewHTTPError(http.StatusForbidden, msg...)
}

func ServerError(msg ...interface{}) error {
	return echo.NewHTTPError(http.StatusInternalServerError, msg...)
}

type flushWriter struct {
	w FlushWriter
}

type FlushWriter interface {
	Write(p []byte) (int, error)
	Flush()
}

func (f *flushWriter) Flush() {
	f.w.Flush()
}

func (f *flushWriter) Write(p []byte) (n int, err error) {
	n, err = f.w.Write(p)
	f.w.Flush()
	return
}

func NewFlushWriter(w FlushWriter) FlushWriter {
	return &flushWriter{w}
}
