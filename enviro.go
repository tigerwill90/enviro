package enviro

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return errors.New("config must be a pointer to a struct")
	}

	val = val.Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("enviro")
		envFormatTag := fieldType.Tag.Get("envformat")

		if tag == "" {
			if field.CanSet() {
				// Handling nested structs or pointers to structs
				if fieldType.Type.Kind() == reflect.Struct || (fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().Kind() == reflect.Struct) {
					nestedStruct := field
					if nestedStruct.Kind() == reflect.Ptr && nestedStruct.IsNil() {
						// Instantiate the nil pointer to a nested struct
						nestedStruct.Set(reflect.New(fieldType.Type.Elem()))
					}

					// Recursively load the nested struct or the newly instantiated struct
					var err error
					if nestedStruct.Kind() == reflect.Ptr {
						err = e.Load(nestedStruct.Interface())
					} else {
						err = e.Load(nestedStruct.Addr().Interface())
					}

					if err != nil {
						return err
					}

					continue
				}
			}

			continue
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
			if err := e.setField(field, envValue, envFormatTag); err != nil {
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

func (e *Enviro) setField(field reflect.Value, value, formatTag string) error {

	// Determine if the field is a pointer and get the element type
	isPtr := field.Type().Kind() == reflect.Ptr
	var elemType reflect.Type
	if isPtr {
		elemType = field.Type().Elem()
	} else {
		elemType = field.Type()
	}

	var err error
	// Create a new value of the element type to hold the converted value
	newVal := reflect.New(elemType).Elem()

	switch elemType.Kind() {
	case reflect.String:
		err = e.setStringField(newVal, value)
	case reflect.Int, reflect.Int32, reflect.Int64:
		err = e.setIntField(newVal, value)
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		err = e.setUintField(newVal, value)
	case reflect.Float32, reflect.Float64:
		err = e.setFloatField(newVal, value)
	case reflect.Bool:
		err = e.setBoolField(newVal, value)
	case reflect.Struct:
		err = e.setStructField(newVal, value, formatTag)
	case reflect.Slice:
		err = e.setSliceField(newVal, value)
	default:
		err = errors.New("unsupported field type")
	}

	if err != nil {
		return err
	}

	// If there was no error and the original field is a pointer, set the field to point to newVal
	if isPtr {
		field.Set(newVal.Addr()) // .Addr() gets the pointer to newVal
	} else {
		field.Set(newVal) // Directly set the value if it's not a pointer
	}

	return nil
}

func (e *Enviro) setStringField(field reflect.Value, value string) error {
	field.SetString(value)
	return nil
}

func (e *Enviro) setIntField(field reflect.Value, value string) error {
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

func (e *Enviro) setUintField(field reflect.Value, value string) error {
	u, err := strconv.ParseUint(value, 10, field.Type().Bits())
	if err != nil {
		return err
	}
	field.SetUint(u)
	return nil
}

func (e *Enviro) setFloatField(field reflect.Value, value string) error {
	f, err := strconv.ParseFloat(value, field.Type().Bits())
	if err != nil {
		return err
	}
	field.SetFloat(f)
	return nil
}

func (e *Enviro) setBoolField(field reflect.Value, value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	field.SetBool(b)
	return nil
}

func (e *Enviro) setSliceField(field reflect.Value, value string) error {
	if field.Type() == reflect.TypeOf([]net.IP(nil)) || field.Type().ConvertibleTo(reflect.TypeOf([]net.IP(nil))) {
		fmt.Println("yolo")
	}

	switch field.Type().Elem().Kind() {
	case reflect.String:
		slice, err := parseStringSlice(value, true)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Int:
		slice, err := parseIntSlice[int](value, 0)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Int8:
		slice, err := parseIntSlice[int8](value, 8)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Int16:
		slice, err := parseIntSlice[int16](value, 16)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Int32:
		slice, err := parseIntSlice[int32](value, 32)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Int64:
		slice, err := parseIntSlice[int64](value, 64)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Uint:
		slice, err := parseUintSlice[uint](value, 0)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Uint8:
		slice, err := parseUintSlice[uint8](value, 8)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Uint16:
		slice, err := parseUintSlice[uint16](value, 16)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Uint32:
		slice, err := parseUintSlice[uint32](value, 32)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Uint64:
		slice, err := parseUintSlice[uint64](value, 64)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Float32:
		slice, err := parseFloatSlice[float32](value, 32)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	case reflect.Float64:
		slice, err := parseFloatSlice[float64](value, 64)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(slice))
	default:
		return fmt.Errorf("unsupported slice element type: %s", field.Type().Elem().Kind().String())
	}
	return nil
}

func (e *Enviro) setStructField(field reflect.Value, value, formatTag string) error {
	if field.Type() == reflect.TypeOf(time.Time{}) || field.Type().ConvertibleTo(reflect.TypeOf(time.Time{})) {
		format, location := parseTimeFormatTag(formatTag)
		return e.setTimeField(field, value, format, location)
	}

	if formatTag == "json" {
		return e.setJsonField(field, value)
	}

	if formatTag == "" {
		formatTag = "-"
	}
	return fmt.Errorf("unsupported format: %q for %s", formatTag, field.Type().String())
}

func (e *Enviro) setTimeField(field reflect.Value, value, format, location string) error {
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

func (e *Enviro) setJsonField(field reflect.Value, value string) error {
	ptrToStruct := reflect.New(field.Type()).Interface()
	if err := json.Unmarshal([]byte(value), &ptrToStruct); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to %s: %w", field.Type().String(), err)
	}
	field.Set(reflect.ValueOf(ptrToStruct).Elem())
	return nil
}
