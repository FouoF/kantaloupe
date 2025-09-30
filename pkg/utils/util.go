package utils

import (
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MarshalObj(obj client.Object) string {
	if obj == nil {
		return ""
	}
	obj.SetManagedFields(nil)
	return StructToString(obj)
}

func StructToString(s interface{}) string {
	data, err := json.Marshal(s)
	if err != nil {
		return err.Error()
	}
	return BytesToString(data)
}

// Compare two maps.
func MapsEqual[
	K comparable,
	V comparable,
](m1, m2 map[K]V) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v := range m1 {
		if val, ok := m2[k]; !ok || v != val {
			return false
		}
	}
	return true
}

func SliceToPointerSlice[T any](slice []T) []*T {
	result := make([]*T, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}
