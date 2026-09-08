package config

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const envPrefix = "TMUS_"

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

type envLookup func(string) (string, bool)

type envBinding struct {
	name  string
	path  string
	value reflect.Value
}

// applyEnv overlays environment variables on a configuration value. Variable
// names are derived from nested TOML paths, prefixed with TMUS_, and uppercased.
func applyEnv(target any, lookup envLookup) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("environment target must be a non-nil pointer to a struct")
	}

	var bindings []envBinding
	if err := collectEnvBindings(value.Elem(), nil, make(map[string]string), &bindings); err != nil {
		return err
	}

	for _, binding := range bindings {
		raw, ok := lookup(binding.name)
		if !ok {
			continue
		}
		if err := setEnvValue(binding.value, raw); err != nil {
			return fmt.Errorf("environment variable %s (%s): %w", binding.name, binding.path, err)
		}
	}

	return nil
}

func collectEnvBindings(value reflect.Value, path []string, names map[string]string, bindings *[]envBinding) error {
	for fieldType, fieldValue := range value.Fields() {
		if !fieldType.IsExported() {
			continue
		}

		name, ok := tomlFieldName(fieldType)
		if !ok {
			continue
		}

		fieldPath := appendPath(path, name)
		if fieldValue.Kind() == reflect.Struct && !implementsTextUnmarshaler(fieldValue) {
			if err := collectEnvBindings(fieldValue, fieldPath, names, bindings); err != nil {
				return err
			}
			continue
		}

		envName := envPrefix + strings.ToUpper(strings.Join(fieldPath, "_"))
		configPath := strings.Join(fieldPath, ".")
		if existingPath, found := names[envName]; found {
			return fmt.Errorf("environment variable %s maps to both %s and %s", envName, existingPath, configPath)
		}
		names[envName] = configPath
		*bindings = append(*bindings, envBinding{name: envName, path: configPath, value: fieldValue})
	}
	return nil
}

func tomlFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("toml")
	name, _, _ := strings.Cut(tag, ",")
	if name == "-" {
		return "", false
	}
	if name == "" {
		name = field.Name
	}
	return name, true
}

func appendPath(path []string, name string) []string {
	next := make([]string, len(path)+1)
	copy(next, path)
	next[len(path)] = name
	return next
}

func implementsTextUnmarshaler(value reflect.Value) bool {
	return value.Type().Implements(textUnmarshalerType) ||
		(value.CanAddr() && value.Addr().Type().Implements(textUnmarshalerType))
}

func setEnvValue(value reflect.Value, raw string) error {
	if value.CanAddr() && value.Addr().Type().Implements(textUnmarshalerType) {
		unmarshaler := value.Addr().Interface().(encoding.TextUnmarshaler)
		if err := unmarshaler.UnmarshalText([]byte(raw)); err != nil {
			return err
		}
		return nil
	}
	if value.Type().Implements(textUnmarshalerType) {
		unmarshaler := value.Interface().(encoding.TextUnmarshaler)
		if err := unmarshaler.UnmarshalText([]byte(raw)); err != nil {
			return err
		}
		return nil
	}

	switch value.Kind() {
	case reflect.String:
		value.SetString(raw)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse boolean: %w", err)
		}
		value.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse integer: %w", err)
		}
		value.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse unsigned integer: %w", err)
		}
		value.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, value.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse number: %w", err)
		}
		value.SetFloat(parsed)
	default:
		return fmt.Errorf("unsupported configuration type %s", value.Type())
	}

	return nil
}
