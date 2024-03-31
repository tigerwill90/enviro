// Copyright 2024 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT License that can be found
// at https://github.com/tigerwill90/enviro/blob/master/LICENSE.txt.

package enviro

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ParseField is an interface that defines how to parse environment variable values.
// Types that implement ParseField can define their own logic to parse the string representation of
// an environment variable into the appropriate Go type.
type ParseField interface {
	// ParseField parses the provided string value and sets the receiver accordingly.
	// It returns an error if the value cannot be parsed into the expected type.
	ParseField(value string) error
}

var parserType = reflect.TypeOf((*ParseField)(nil)).Elem()

// Enviro facilitates the loading and parsing of environment variables into Go structs.
// It supports custom prefixes for environment variables, nested struct parsing, and fields of various types.
type Enviro struct {
	prefix string
}

// New creates and returns a new instance of the Enviro parser.
func New() *Enviro {
	return &Enviro{}
}

// SetEnvPrefix sets a custom prefix that will be prepended to all environment variable names
// when parsing. Fields with the `enviro:your_var_name,omitprefix` will ignore the prefix.
func (e *Enviro) SetEnvPrefix(prefix string) {
	e.prefix = prefix
}

// ParseEnvWithPrefix parses environment variables into the provided struct based on struct tags.
// It uses the specified prefix to look up environment variables, allowing for nested struct parsing
// and the application of custom parsing logic for specific fields. The function returns an error
// if parsing fails for any field, or if the provided `config` is not a pointer to a struct.
//
// The `config` parameter should be a pointer to the struct you wish to populate with environment
// variable values. If the struct contains nested structs and the tag `enviro:"nested:your_prefix"`, the prefix is
// concatenated with "_" and the nested struct's tag to form the complete environment variable name.
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
		envOpt := fieldType.Tag.Get("envopt")
		envDef := fieldType.Tag.Get("envdefault")

		if tag == "" || strings.HasPrefix(tag, "nested:") {
			if field.CanSet() {
				// Handling nested structs or pointers to structs
				if fieldType.Type.Kind() == reflect.Struct || (fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().Kind() == reflect.Struct) {
					nestedStruct := field
					if nestedStruct.Kind() == reflect.Ptr && nestedStruct.IsNil() {
						// Instantiate the nil pointer to a nested struct
						nestedStruct.Set(reflect.New(fieldType.Type.Elem()))
					}

					var envPrefix string
					if prefix != "" {
						envPrefix = prefix + "_"
					}
					envPrefix += strings.TrimPrefix(tag, "nested:")

					// Recursively load the nested struct or the newly instantiated struct
					var err error
					if nestedStruct.Kind() == reflect.Ptr {
						err = e.ParseEnvWithPrefix(nestedStruct.Interface(), envPrefix)
					} else {
						err = e.ParseEnvWithPrefix(nestedStruct.Addr().Interface(), envPrefix)
					}

					if err != nil {
						return err
					}

					continue
				}
			}

			continue
		}

		envKey, omitprefix, required := parseTag(tag)
		if !omitprefix && prefix != "" {
			envKey = prefix + "_" + envKey
		}

		envValue, exists := os.LookupEnv(strings.ToUpper(envKey))
		if required && !exists {
			return fmt.Errorf("missing required environment variable: %s", strings.ToUpper(envKey))
		}
		if required && envValue == "" {
			return fmt.Errorf("empty required environment variable: %s", strings.ToUpper(envKey))
		}

		if envValue == "" {
			envValue = envDef
		}

		if exists || envValue != "" {
			if err := e.setField(field, envValue, envOpt); err != nil {
				return fmt.Errorf("failed to parse environment variable %s: %w", strings.ToUpper(envKey), err)
			}
		}
	}
	return nil
}

// ParseEnv is a convenience method that calls ParseEnvWithPrefix with the base prefix set on the Enviro
// instance.
func (e *Enviro) ParseEnv(config any) error {
	return e.ParseEnvWithPrefix(config, e.prefix)
}

// MustParseEnv is a convenience method that calls ParseEnv and panics if an error occurs.
func (e *Enviro) MustParseEnv(config any) {
	if err := e.ParseEnv(config); err != nil {
		panic(err)
	}
}

func parseTag(tag string) (key string, omitprefix, required bool) {
	parts := strings.Split(tag, ",")
	key = strings.TrimSpace(parts[0])
	if len(parts) > 2 {
		if strings.TrimSpace(parts[1]) == "required" {
			required = true
		}
		if strings.TrimSpace(parts[2]) == "omitprefix" {
			omitprefix = true
		}
		return
	}

	if len(parts) > 1 {
		if strings.TrimSpace(parts[1]) == "required" {
			required = true
		}
		if strings.TrimSpace(parts[1]) == "omitprefix" {
			omitprefix = true
		}
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

func parseFileFormatTag(tag string) (flag int, perm os.FileMode) {
	// Default to read-only if no specific options are provided
	flag = os.O_RDONLY // Default flag
	perm = 0666        // Default permission for new files

	if strings.HasPrefix(tag, "file:") {
		options := strings.TrimPrefix(tag, "file:")
		parts := strings.Split(options, ",")

		// Assume the first part specifies flags and the second part specifies permissions
		for _, part := range parts {
			part = strings.TrimSpace(part)
			opts := strings.Split(part, "|")
			for _, opt := range opts {
				switch strings.TrimSpace(opt) {
				case "ro":
					flag = os.O_RDONLY
				case "wo":
					flag = os.O_WRONLY
				case "rw":
					flag = os.O_RDWR
				case "create":
					flag |= os.O_CREATE
				case "truncate":
					flag |= os.O_TRUNC
				case "append":
					flag |= os.O_APPEND
				default:
					// Attempt to parse permission if it's not a known flag
					if permValue, err := strconv.ParseUint(opt, 8, 32); err == nil {
						perm = os.FileMode(permValue)
					}
				}
			}
		}
	}
	return flag, perm
}

func (e *Enviro) setField(field reflect.Value, value, opt string) error {

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
		err = e.setStructField(target, value, opt)
	case reflect.Slice:
		err = e.setSliceField(target, value, opt)
	case reflect.Map:
		err = e.setMapField(target, value, opt)
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

func (e *Enviro) setSliceField(field reflect.Value, value, opt string) error {
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
			if err := e.setSliceField(newVal, elem, opt); err != nil {
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
			if err := e.setStructField(newVal, elem, opt); err != nil {
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

func (e *Enviro) setStructField(field reflect.Value, value, opt string) error {

	switch field.Type() {
	case reflect.TypeOf(time.Time{}):
		format, location := parseTimeFormatTag(opt)
		return e.setTimeField(field, value, format, location)
	case reflect.TypeOf(time.Location{}):
		loc, err := time.LoadLocation(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(*loc))
		return nil
	case reflect.TypeOf(url.URL{}):
		u, err := url.Parse(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(*u))
		return nil
	case reflect.TypeOf(os.File{}):
		flag, perm := parseFileFormatTag(opt)
		return e.setFileField(field, value, flag, perm)
	}

	switch opt {
	case "json":
		return e.setJsonField(field, value)
	case "yaml":
		return e.setYamlField(field, value)
	}

	if opt == "" {
		opt = "-"
	}
	return fmt.Errorf("unsupported format %q for %s", opt, field.Type().String())
}

func (e *Enviro) setMapField(field reflect.Value, value, opt string) error {
	switch opt {
	case "json":
		return e.setJsonField(field, value)
	case "yaml":
		return e.setYamlField(field, value)
	}

	if opt == "" {
		opt = "-"
	}
	return fmt.Errorf("unsupported format %q for %s", opt, field.Type().String())
}

func (e *Enviro) setTimeField(field reflect.Value, value, format, location string) error {
	loc := time.UTC
	if location != "" {
		var err error
		loc, err = time.LoadLocation(location)
		if err != nil {
			return err
		}
	}

	if format != "" {
		t, err := time.ParseInLocation(format, value, loc)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(t))
		return nil
	}

	t, err := parseDateWith(value, timeFormats, loc)
	if err != nil {
		return err
	}
	field.Set(reflect.ValueOf(t))
	return nil
}

func (e *Enviro) setFileField(field reflect.Value, value string, flag int, perm os.FileMode) error {
	f, err := os.OpenFile(value, flag, perm)
	if err != nil {
		return err
	}
	field.Set(reflect.ValueOf(*f))
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

func (e *Enviro) setYamlField(field reflect.Value, value string) error {
	var addr reflect.Value
	if field.Kind() == reflect.Ptr && !field.IsNil() {
		// Field is a non-nil pointer, so we can work directly with its element
		addr = field
	} else if field.CanAddr() {
		// Field is addressable (but not a pointer), so we get its address
		addr = field.Addr()
	} else {
		// Field is neither a non-nil pointer nor addressable; this should never happen.
		return fmt.Errorf("failed to unmarshal YAML to %s: field is not addressable", field.Type().String())
	}

	// Unmarshal YAML into the addressable field or the element pointed to by the field
	if err := yaml.Unmarshal([]byte(value), addr.Interface()); err != nil {
		return fmt.Errorf("failed to unmarshal YAML to %s: %w", field.Type().String(), err)
	}

	return nil
}
