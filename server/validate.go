package server

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/knight42/Yuki/cron"
	"gopkg.in/go-playground/validator.v9"
)

func isNum(k reflect.Kind) bool {
	return k == reflect.Float32 ||
		k == reflect.Float64 ||
		k == reflect.Int ||
		k == reflect.Int8 ||
		k == reflect.Int32 ||
		k == reflect.Int64 ||
		k == reflect.Uint ||
		k == reflect.Uint8 ||
		k == reflect.Uint32 ||
		k == reflect.Uint64
}

func getJSONName(tag string) (name string, hasName bool) {
	specs := strings.Split(tag, ",")
	hasName = false
	for _, s := range specs {
		switch s {
		case "omitempty":

		case "":
			fallthrough
		case "-":
			return "", false

		default:
			return s, true
		}
	}
	return "", false
}

type myValidator struct {
	v *validator.Validate
}

func (mv *myValidator) Validate(i interface{}) error {
	return mv.v.Struct(i)
}

func (mv *myValidator) CheckMap(m map[string]interface{}, i interface{}) error {
	ty := reflect.ValueOf(i).Type()
	if ty.Kind() != reflect.Struct {
		return fmt.Errorf("not a struct: %s", ty.Name())
	}
	fields := ty.NumField()
	fieldMap := make(map[string]reflect.StructField)
	for i := 0; i < fields; i++ {
		field := ty.Field(i)
		if name, ok := getJSONName(field.Tag.Get("json")); ok {
			fieldMap[name] = field
		}
	}
	for k, v := range m {
		if strings.HasPrefix(k, "envs.") {
			continue
		}
		if strings.HasPrefix(k, "volumes.") {
			if reflect.ValueOf(v).Kind() != reflect.String {
				return fmt.Errorf("not a string: %v", v)
			}
			continue
		}
		field, ok := fieldMap[k]
		if !ok {
			return fmt.Errorf("unexpected key: %s", k)
		}
		expectedKind := field.Type.Kind()
		actualKind := reflect.ValueOf(v).Kind()

		// try converting string to float
		if isNum(expectedKind) && actualKind == reflect.String {
			s := v.(string)
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				m[k] = f
				continue
			}
		}
		if actualKind == reflect.Float64 && isNum(expectedKind) {
			// do nothing
		} else if actualKind != expectedKind {
			return fmt.Errorf("mismatched kind of <%s>, expect %s, get %s", k, expectedKind, actualKind)
		}

		if rule, ok := field.Tag.Lookup("validate"); ok {
			if err := mv.v.Var(v, rule); err != nil {
				return fmt.Errorf("invalid key: %s", k)
			}
		}
	}
	return nil
}

func NewValidator() *validator.Validate {
	v := validator.New()
	v.RegisterValidation("mongodb", isMongoDB)
	v.RegisterValidation("cron", isCron)
	v.RegisterValidation("hostport", isHostPort)
	v.RegisterValidation("duration", isDuration)
	return v
}

func isDuration(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	dur, err := time.ParseDuration(s)
	if err != nil {
		return false
	}
	return dur >= 0
}

func isMongoDB(fl validator.FieldLevel) bool {
	url := fl.Field().String()
	_, err := mgo.ParseURL(url)
	return err == nil
}

func isCron(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	_, err := cron.Parse(s)
	return err == nil
}

func isHostPort(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	_, err := net.ResolveTCPAddr("tcp", s)
	return err == nil
}
