package utils

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FieldNameMapper func(apiField string) (goField string, ok bool)

// SortStructSlice using the value of fieldPath specifid go struct field
// to sort the slice. FieldPath can be nested, e.g. "Profile.CreateTime".
// "mapper" specify how to map the field name to the go field name, nil
// means no mapping.
func SortStructSlice(slice any, fieldPath string, ascending bool, mapper FieldNameMapper) error {
	if slice == nil || fieldPath == "" {
		return nil
	}
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice type, got %s", v.Kind())
	}
	if v.Len() == 0 {
		return nil
	}

	path := parseAndMapFieldPath(fieldPath, mapper)

	sort.Slice(slice, func(i, j int) bool {
		vi, err := resolveNestedField(reflect.Indirect(v.Index(i)), path)
		if err != nil {
			panic(err)
		}
		vj, err := resolveNestedField(reflect.Indirect(v.Index(j)), path)
		if err != nil {
			panic(err)
		}

		less, err := compareValues(vi, vj)
		if err != nil {
			panic(err)
		}
		if ascending {
			return less
		}
		return !less
	})

	return nil
}

func parseAndMapFieldPath(fieldPath string, mapper FieldNameMapper) []string {
	parts := strings.Split(fieldPath, ".")
	for i, part := range parts {
		if mapper != nil {
			if mapped, ok := mapper(part); ok {
				parts[i] = mapped
			}
		}
	}
	return parts
}

func resolveNestedField(v reflect.Value, path []string) (reflect.Value, error) {
	for _, fieldName := range path {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}, fmt.Errorf("nil pointer when accessing '%s'", fieldName)
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("expect struct when accessing '%s', got %s", fieldName, v.Kind())
		}
		v = v.FieldByName(fieldName)
		if !v.IsValid() {
			return reflect.Value{}, fmt.Errorf("field '%s' not found", fieldName)
		}
	}
	return v, nil
}

func compareValues(vi, vj reflect.Value) (bool, error) {
	switch vi.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vi.Int() < vj.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vi.Uint() < vj.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return vi.Float() < vj.Float(), nil
	case reflect.Bool:
		return !vi.Bool(), nil
	case reflect.String:
		return vi.String() < vj.String(), nil
	case reflect.Struct:
		if vi.Type() == reflect.TypeOf(time.Time{}) {
			t1 := vi.Interface().(time.Time)
			t2 := vj.Interface().(time.Time)
			return t1.Before(t2), nil
		}
		if vi.Type() == reflect.TypeOf(metav1.Time{}) {
			t1 := vi.Interface().(metav1.Time).Time
			t2 := vj.Interface().(metav1.Time).Time
			return t1.Before(t2), nil
		}
	}
	return false, fmt.Errorf("unsupported field type: %s", vi.Type())
}

// Map Snake name to Camel name, suitable for proto to go struct.
func SnakeToCamelMapper() FieldNameMapper {
	return func(field string) (string, bool) {
		if len(field) == 0 {
			return "", false
		}
		parts := strings.Split(field, "_")
		for i, part := range parts {
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		return strings.Join(parts, ""), true
	}
}

// Manually map field name to go field name.
func StaticMapper(mapping map[string]string) FieldNameMapper {
	return func(field string) (string, bool) {
		v, ok := mapping[field]
		return v, ok
	}
}

// Using static mapping and snake to camel mapping together.
func CombinedMapper(static map[string]string) FieldNameMapper {
	snakeMapper := SnakeToCamelMapper()
	return func(field string) (string, bool) {
		if v, ok := static[field]; ok {
			return v, true
		}
		return snakeMapper(field)
	}
}
