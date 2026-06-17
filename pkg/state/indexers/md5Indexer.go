package indexers

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"strings"
)

// MD5FieldsIndexer is used to create an index based on md5sum of
// string or *string fields.
type MD5FieldsIndexer struct {
	// Fields to use for md5sum calculation
	Fields []string
}

// FromObject take Obj and returns index key formed using
// the fields.
func (s *MD5FieldsIndexer) FromObject(obj any) (bool, []byte, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	var blob strings.Builder
	for _, field := range s.Fields {
		fv := v.FieldByName(field)
		fv = reflect.Indirect(fv)
		if !fv.IsValid() {
			return false, nil,
				fmt.Errorf("field '%s' for %#v is invalid", field, obj)
		}
		blob.WriteString(fv.String())
	}
	if blob.String() == "" {
		return false, nil, nil
	}
	md5Sum := md5.Sum([]byte(blob.String()))
	return true, md5Sum[:], nil
}

// FromArgs takes in a string and returns its byte form.
func (s *MD5FieldsIndexer) FromArgs(args ...any) ([]byte, error) {
	var blob strings.Builder
	for _, arg := range args {
		s, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("argument must be a string: %#v", arg)
		}
		blob.WriteString(s)
	}
	if blob.String() == "" {
		return nil, fmt.Errorf("empty args is not a valid value")
	}
	// Add the null character as a terminator
	md5Sum := md5.Sum([]byte(blob.String()))
	return md5Sum[:], nil
}
