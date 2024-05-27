package utils

import (
	"fmt"
	"reflect"

	"github.com/kong/go-kong/kong"
)

// MustMergeTags is same as MergeTags but panics if there is an error.
func MustMergeTags(obj interface{}, tags []string) {
	err := MergeTags(obj, tags)
	if err != nil {
		panic(err)
	}
}

// MergeTags merges Tags in the object with tags.
func MergeTags(obj interface{}, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	ptr := reflect.ValueOf(obj)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("obj is not a pointer")
	}
	v := reflect.Indirect(ptr)
	structTags := v.FieldByName("Tags")
	var zero reflect.Value
	if structTags == zero {
		return nil
	}
	m := make(map[string]bool)
	for i := 0; i < structTags.Len(); i++ {
		tag := reflect.Indirect(structTags.Index(i)).String()
		m[tag] = true
	}
	for _, tag := range tags {
		if _, ok := m[tag]; !ok {
			t := tag
			structTags.Set(reflect.Append(structTags, reflect.ValueOf(&t)))
		}
	}
	return nil
}

// MustRemoveTags is same as RemoveTags but panics if there is an error.
func MustRemoveTags(obj interface{}, tags []string) {
	err := RemoveTags(obj, tags)
	if err != nil {
		panic(err)
	}
}

// RemoveTags removes tags from the Tags in obj.
func RemoveTags(obj interface{}, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	m := make(map[string]bool)
	for _, tag := range tags {
		m[tag] = true
	}

	ptr := reflect.ValueOf(obj)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("obj is not a pointer")
	}
	v := reflect.Indirect(ptr)
	structTags := v.FieldByName("Tags")
	var zero reflect.Value
	if structTags == zero {
		return nil
	}

	res := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(reflect.TypeOf(""))), 0, 0)
	for i := 0; i < structTags.Len(); i++ {
		tag := reflect.Indirect(structTags.Index(i)).String()
		if !m[tag] {
			res = reflect.Append(res, structTags.Index(i))
		}
	}
	structTags.Set(res)
	return nil
}

// HasTags checks if the given object has any of the specified tags.
// The function returns true if at least one of the provided tags is present in the object's tags.
func HasTags[T *kong.Consumer](obj T, tags []string) bool {
	if len(tags) == 0 {
		return true
	}

	m := make(map[string]struct{})
	for _, tag := range tags {
		m[tag] = struct{}{}
	}

	switch obj := any(obj).(type) {
	case *kong.Consumer:
		for _, tag := range obj.Tags {
			if tag == nil {
				continue
			}
			if _, ok := m[*tag]; ok {
				return true
			}
		}
	default:
		return false
	}
	return false
}
