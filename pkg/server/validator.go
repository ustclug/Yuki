package server

type echoValidator func(i any) error

func (v echoValidator) Validate(i any) error {
	return v(i)
}
