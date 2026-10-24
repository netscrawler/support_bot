package pkg

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func GenerateEnv(v any, prefix, groupComment string) (string, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return "", errors.New("expected struct")
	}

	rt := rv.Type()

	var b strings.Builder

	if groupComment != "" {
		for line := range strings.SplitSeq(groupComment, "\n") {
			b.WriteString("# " + line + "\n")
		}

		b.WriteString("\n")
	}

	for i := range rv.NumField() {
		field := rt.Field(i)
		fv := rv.Field(i)

		if field.PkgPath != "" {
			continue
		}

		envKey := field.Tag.Get("env")
		comment := field.Tag.Get("comment")
		def := field.Tag.Get("env-default")

		if fv.Kind() == reflect.Struct {
			subPrefix := prefix

			if envKey != "" {
				if subPrefix != "" {
					subPrefix += "_"
				}

				subPrefix += envKey
			}

			subGroupComment := comment

			subEnv, err := GenerateEnv(
				fv.Interface(),
				subPrefix,
				subGroupComment,
			)
			if err != nil {
				return "", err
			}

			b.WriteString(subEnv)
			b.WriteString("\n")

			continue
		}

		if envKey == "" {
			continue
		}

		if prefix != "" {
			envKey = prefix + "_" + envKey
		}

		value := def
		if value == "" {
			if fv.Kind() == reflect.String {
				value = fv.String() // не Sprint: secret.String печатается как "***"
			} else {
				value = fmt.Sprint(fv.Interface())
			}
		}

		if comment != "" {
			for line := range strings.SplitSeq(comment, "\n") {
				b.WriteString("# " + line + "\n")
			}
		}

		if fv.Kind() == reflect.Slice {
			sliceVals := make([]string, fv.Len())
			for j := range fv.Len() {
				sliceVals[j] = fmt.Sprint(fv.Index(j).Interface())
			}

			value = "[" + strings.Join(sliceVals, ",") + "]"
		}

		b.WriteString(envKey + "=" + value + "\n\n")
	}

	return b.String(), nil
}
