package server

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

type echoValidator func(i any) error

func (v echoValidator) Validate(i any) error {
	return v(i)
}

func InitValidator() *validator.Validate {
	validate := validator.New()
	validate.RegisterValidation("repo-name", func(fl validator.FieldLevel) bool {
		// Avoid possible issues when using filepath.Join with repo name
		field := fl.Field().String()
		return !strings.Contains(field, "/") && field != ".."
	})
	return validate
}
