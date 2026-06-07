package server

import (
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/ustclug/Yuki/pkg/controlplane"
)

type echoValidator func(i any) error

func (v echoValidator) Validate(i any) error {
	return v(i)
}

func InitValidator() *validator.Validate {
	validate := validator.New()
	_ = validate.RegisterValidation("repo-name", func(fl validator.FieldLevel) bool {
		// Avoid possible issues when using filepath.Join with repo name
		field := fl.Field().String()
		return !strings.Contains(field, "/") && field != ".."
	})
	_ = validate.RegisterValidation("listen-addr", func(fl validator.FieldLevel) bool {
		_, err := controlplane.ParseEndpoint(fl.Field().String())
		return err == nil
	})
	return validate
}
