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

type ParseField interface {
	ParseField(value string) error
}

var parserType = reflect.TypeOf((*ParseField)(nil)).Elem()

type Enviro struct {
	prefix string
}

func New() *Enviro {
	return &Enviro{}
}

func (e *Enviro) SetEnvPrefix(prefix string) {
	e.prefix = prefix
}

func (e *Enviro) ParseEnvWithPrefix(config any, prefix string) error {
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

		if tag == "" || strings.HasPrefix(tag, "prefix:") {
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
						err = e.ParseEnvWithPrefix(nestedStruct.Interface(), prefix+"_"+strings.TrimPrefix(tag, "prefix:"))
					} else {
						err = e.ParseEnvWithPrefix(nestedStruct.Addr().Interface(), prefix+"_"+strings.TrimPrefix(tag, "prefix:"))
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
		if prefix != "" {
			envKey = prefix + "_" + envKey
		}

		envValue, exists := os.LookupEnv(strings.ToUpper(envKey))
		if !exists && required {
			return fmt.Errorf("missing required environment variable: %s", strings.ToUpper(envKey))
		}

		if exists {
			if err := e.setField(field, envValue, envFormatTag); err != nil {
				return fmt.Errorf("failed to parse environment variable %s: %w", strings.ToUpper(envKey), err)
			}
		}
	}
	return nil
}

func (e *Enviro) ParseEnv(config any) error {
	return e.ParseEnvWithPrefix(config, e.prefix)
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

	var target reflect.Value
	if isPtr {
		if field.IsNil() {
			// Create a new value of the element type to hold the converted value
			target = reflect.New(elemType).Elem()
		} else {
			target = field.Elem()
		}
	} else {
		target = field
	}

	var err error
	// Check if the type implements the ParseField interface
	if target.Addr().Type().Implements(parserType) {
		// The field implements ParseField interface, delegate parsing to it
		parser := target.Addr().Interface().(ParseField)
		err = parser.ParseField(value)
		goto SET_FIELD
	}

	switch elemType.Kind() {
	case reflect.String:
		err = e.setStringField(target, value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		err = e.setIntField(target, value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		err = e.setUintField(target, value)
	case reflect.Float32, reflect.Float64:
		err = e.setFloatField(target, value)
	case reflect.Bool:
		err = e.setBoolField(target, value)
	case reflect.Struct:
		err = e.setStructField(target, value, formatTag)
	case reflect.Slice:
		err = e.setSliceField(target, value, formatTag)
	case reflect.Map:
		err = e.setMapField(target, value, formatTag)
	default:
		err = errors.New("unsupported field type")
	}

	//goland:noinspection GoSnakeCaseUsage
SET_FIELD:
	if err != nil {
		return err
	}

	// If there was no error and the original field is a pointer, set the field to point to target
	if isPtr {
		field.Set(target.Addr()) // .Addr() gets the pointer to target
	} else {
		field.Set(target) // Directly set the value if it's not a pointer
	}

	return nil
}

func (e *Enviro) setStringField(field reflect.Value, value string) error {
	field.Set(reflect.ValueOf(value))
	return nil
}

func (e *Enviro) setIntField(field reflect.Value, value string) error {
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
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

func (e *Enviro) setSliceField(field reflect.Value, value, formatTag string) error {
	elements := strings.Split(value, ",")
	slice := reflect.MakeSlice(field.Type(), len(elements), len(elements))

	isPtr := field.Type().Elem().Kind() == reflect.Ptr
	var elemTyp reflect.Type
	if isPtr {
		elemTyp = field.Type().Elem().Elem()
	} else {
		elemTyp = field.Type().Elem()
	}

	if slice.Index(0).Addr().Type().Implements(parserType) || slice.Index(0).Type().Implements(parserType) {
		for i, elem := range elements {
			newVal := reflect.New(elemTyp).Elem()
			parser := newVal.Addr().Interface().(ParseField)
			if err := parser.ParseField(elem); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(newVal.Addr())
			} else {
				slice.Index(i).Set(newVal)
			}
		}
		field.Set(reflect.AppendSlice(field, slice))
		return nil
	}

	switch elemTyp.Kind() {
	case reflect.String:
		for i, elem := range elements {
			newVal := reflect.New(elemTyp).Elem()
			if err := e.setStringField(newVal, strings.TrimSpace(elem)); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(newVal.Addr())
			} else {
				slice.Index(i).Set(newVal)
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		for i, elem := range elements {
			newVal := reflect.New(elemTyp).Elem()
			if err := e.setIntField(newVal, strings.TrimSpace(elem)); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(newVal.Addr())
			} else {
				slice.Index(i).Set(newVal)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Type() == reflect.TypeOf(net.IP(nil)) {
			ip := net.ParseIP(value)
			if ip == nil {
				return errors.New("invalid IP address")
			}
			field.Set(reflect.ValueOf(ip))
			return nil
		}

		if field.Type() == reflect.TypeOf([]net.HardwareAddr(nil)) {
			addr, err := net.ParseMAC(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(addr))
			return nil
		}

		for i, elem := range elements {
			newVal := reflect.New(elemTyp).Elem()
			if err := e.setUintField(newVal, strings.TrimSpace(elem)); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(newVal.Addr())
			} else {
				slice.Index(i).Set(newVal)
			}
		}
	case reflect.Float32, reflect.Float64:
		for i, elem := range elements {
			newVal := reflect.New(elemTyp).Elem()
			if err := e.setFloatField(newVal, strings.TrimSpace(elem)); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(newVal.Addr())
			} else {
				slice.Index(i).Set(newVal)
			}
		}
	case reflect.Bool:
		for i, elem := range elements {
			newVal := reflect.New(elemTyp).Elem()
			if err := e.setBoolField(newVal, strings.TrimSpace(elem)); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(newVal.Addr())
			} else {
				slice.Index(i).Set(newVal)
			}
		}
	case reflect.Slice:
		for i, elem := range elements {
			newVal := reflect.New(elemTyp).Elem()
			if err := e.setSliceField(newVal, elem, formatTag); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(newVal.Addr())
			} else {
				slice.Index(i).Set(newVal)
			}
		}
	case reflect.Struct:
		for i, elem := range elements {
			newVal := reflect.New(elemTyp).Elem()
			if err := e.setStructField(newVal, elem, formatTag); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(newVal.Addr())
			} else {
				slice.Index(i).Set(newVal)
			}
		}
	default:
		return fmt.Errorf("unsupported slice element type: %s", elemTyp.String())
	}

	field.Set(reflect.AppendSlice(field, slice))
	return nil
}

func (e *Enviro) setStructField(field reflect.Value, value, formatTag string) error {
	if field.Type() == reflect.TypeOf(time.Time{}) {
		format, location := parseTimeFormatTag(formatTag)
		return e.setTimeField(field, value, format, location)
	}

	if field.Type() == reflect.TypeOf(time.Location{}) {
		loc, err := time.LoadLocation(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(*loc))
		return nil
	}

	switch formatTag {
	case "json":
		return e.setJsonField(field, value)
	}

	if formatTag == "" {
		formatTag = "-"
	}
	return fmt.Errorf("unsupported format %q for %s", formatTag, field.Type().String())
}

func (e *Enviro) setMapField(field reflect.Value, value, formatTag string) error {
	switch formatTag {
	case "json":
		return e.setJsonField(field, value)
	}

	if formatTag == "" {
		formatTag = "-"
	}
	return fmt.Errorf("unsupported format %q for %s", formatTag, field.Type().String())
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
		field.Set(reflect.ValueOf(t))
		return nil
	}

	t, err := parseDateWith(value, time.UTC, timeFormats)
	if err != nil {
		return err
	}
	field.Set(reflect.ValueOf(t))
	return nil
}

func (e *Enviro) setJsonField(field reflect.Value, value string) error {
	v := reflect.New(field.Type()).Interface()
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to %s: %w", field.Type().String(), err)
	}
	field.Set(reflect.ValueOf(v).Elem())
	return nil
}
