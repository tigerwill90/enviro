package enviro

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Enviro struct {
	prefix string
}

func New() *Enviro {
	return &Enviro{}
}

func (e *Enviro) SetEnvPrefix(prefix string) {
	e.prefix = prefix
}

func (e *Enviro) Load(config any) error {
	val := reflect.ValueOf(config).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("enviro")
		envFormatTag := fieldType.Tag.Get("envformat")

		if tag == "" {
			continue // Skip fields without `enviro` tag
		}

		envKey, required := parseTag(tag)
		if e.prefix != "" {
			envKey = e.prefix + "_" + envKey
		}

		envValue, exists := os.LookupEnv(strings.ToUpper(envKey))
		if !exists && required {
			return fmt.Errorf("missing required environment variable: %s", strings.ToUpper(envKey))
		}

		if exists {
			if err := setField(field, envValue, envFormatTag); err != nil {
				return err
			}
		}
	}

	return nil
}

func parseTag(tag string) (key string, required bool) {
	parts := strings.Split(tag, ",")
	key = strings.TrimSpace(parts[0])
	if len(parts) > 1 && strings.TrimSpace(parts[1]) == "required" {
		required = true
	}
	return
}

func parseTimeFormatTag(tag string) (format, location string) {
	if strings.HasPrefix(tag, "time:") {
		parts := strings.Split(strings.TrimPrefix(tag, "time:"), ",")
		format = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			location = strings.TrimSpace(parts[1])
		}
	}
	return
}

// From html/template/content.go
// Copyright 2011 The Go Authors. All rights reserved.
// indirect returns the value, after dereferencing as many times
// as necessary to reach the base type (or nil).
func indirect(a any) any {
	if a == nil {
		return nil
	}
	if t := reflect.TypeOf(a); t.Kind() != reflect.Ptr {
		// Avoid creating a reflect.Value if it's not a pointer.
		return a
	}
	v := reflect.ValueOf(a)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
}

func setField(field reflect.Value, value, formatTag string) error {

	switch field.Kind() {
	case reflect.String:
		return setStringField(field, value)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return setIntField(field, value)
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		return setUintField(field, value)
	case reflect.Float32, reflect.Float64:
		return setFloatField(field, value)
	case reflect.Bool:
		return setBoolField(field, value)
	case reflect.Struct:
		return setStructField(field, value, formatTag)
	case reflect.Slice:
		return setSliceField(field, value)
	default:
		return errors.New("unsupported field type")
	}
}

func setStringField(field reflect.Value, value string) error {
	field.SetString(value)
	return nil
}

func setIntField(field reflect.Value, value string) error {
	if field.Type() == reflect.TypeOf(time.Duration(0)) || field.Type().ConvertibleTo(reflect.TypeOf(time.Duration(0))) {
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		field.SetInt(int64(d))
		return nil
	}
	i, err := strconv.ParseInt(value, 10, field.Type().Bits())
	if err != nil {
		return err
	}

	field.SetInt(i)
	return nil
}

func setUintField(field reflect.Value, value string) error {
	u, err := strconv.ParseUint(value, 10, field.Type().Bits())
	if err != nil {
		return err
	}
	field.SetUint(u)
	return nil
}

func setFloatField(field reflect.Value, value string) error {
	f, err := strconv.ParseFloat(value, field.Type().Bits())
	if err != nil {
		return err
	}
	field.SetFloat(f)
	return nil
}

func setBoolField(field reflect.Value, value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	field.SetBool(b)
	return nil
}

func setSliceField(field reflect.Value, value string) error {
	switch field.Type().Elem().Kind() {
	case reflect.String:
		slice, err := parseStringSlice(value, true)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Int:
		slice, err := parseIntSlice(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	default:
		return fmt.Errorf("unsupported slice element type: %s", field.Type().Elem().Kind().String())
	}
	return nil
}

func setStructField(field reflect.Value, value, formatTag string) error {
	if field.Type() == reflect.TypeOf(time.Time{}) || field.Type().ConvertibleTo(reflect.TypeOf(time.Time{})) {
		format, location := parseTimeFormatTag(formatTag)
		return setTimeField(field, value, format, location)
	}

	if formatTag == "json" {
		return setJsonField(field, value)
	}

	return fmt.Errorf("unsupported struct type: %s", field.Type().String())
}

func setTimeField(field reflect.Value, value, format, location string) error {
	if format != "" {
		loc := time.UTC
		if location != "" {
			var err error
			loc, err = time.LoadLocation(location)
			if err != nil {
				return err
			}
		}

		t, err := time.ParseInLocation(format, value, loc)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(t).Convert(field.Type()))
		return nil
	}

	t, err := parseDateWith(value, time.UTC, timeFormats)
	if err != nil {
		return err
	}
	field.Set(reflect.ValueOf(t).Convert(field.Type()))
	return nil
}

func setJsonField(field reflect.Value, value string) error {
	ptrToStruct := reflect.New(field.Type()).Interface()
	if err := json.Unmarshal([]byte(value), &ptrToStruct); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to %s: %w", field.Type().String(), err)
	}
	field.Set(reflect.ValueOf(ptrToStruct).Elem())
	return nil
}
