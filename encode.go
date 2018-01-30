package bencode

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strconv"
)

func Marshal(data interface{}) (result []byte, err error) {
	defer handlePanic(&err)
	ms := new(marshalState)
	ms.marshal(reflect.ValueOf(data))
	result = ms.Bytes()
	return
}

type marshalState struct {
	bytes.Buffer
	scratch [64]byte
}

func (ms *marshalState) marshal(data reflect.Value) {
	switch data.Kind() {
	case reflect.String:
		ms.marshalBytes([]byte(data.String()))
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		ms.marshalInt64(data.Int())
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		ms.marshalUint64(data.Uint())
	case reflect.Slice:
		if data.Type().Elem().Kind() == reflect.Uint8 {
			ms.marshalBytes(data.Bytes())
		} else {
			ms.marshalList(data)
		}
	case reflect.Array:
		ms.marshalList(data)
	case reflect.Map:
		keyKind := data.Type().Key().Kind()
		if keyKind != reflect.String {
			panic(fmt.Errorf("cannot unmarshal map with key type %t", data.Interface()))
		}
		ms.marshalMap(data)
	case reflect.Struct:
		ms.marshalStruct(data)
	case reflect.Interface:
		ms.marshal(data.Elem())
	default:
		panic(fmt.Errorf("err: %v has unsupported type %v",
			data.Interface(), data.Kind()))
	}
}

func (ms *marshalState) marshalBytes(bs []byte) {
	b := strconv.AppendInt(ms.scratch[:0], int64(len(bs)), 10)
	ms.Write(b)
	ms.WriteByte(':')
	ms.Write(bs)
}

func (ms *marshalState) marshalInt64(n int64) {
	ms.WriteByte('i')
	b := strconv.AppendInt(ms.scratch[:0], n, 10)
	ms.Write(b)
	ms.WriteByte('e')
}

func (ms *marshalState) marshalUint64(n uint64) {
	ms.WriteByte('i')
	b := strconv.AppendUint(ms.scratch[:0], n, 10)
	ms.Write(b)
	ms.WriteByte('e')
}

func (ms *marshalState) marshalList(l reflect.Value) {
	ms.WriteByte('l')
	for i := 0; i < l.Len(); i++ {
		ms.marshal(l.Index(i))
	}
	ms.WriteByte('e')
}

func (ms *marshalState) marshalMap(m reflect.Value) {
	ms.WriteByte('d')
	// bencoding specification states that the keys must be sorted
	keys := m.MapKeys()
	sort.Slice(
		keys,
		func(i, j int) bool { return keys[i].String() < keys[j].String() },
	)
	for _, k := range keys {
		ms.marshalBytes([]byte(k.String()))
		ms.marshal(m.MapIndex(k))
	}
	ms.WriteByte('e')
}

func (ms *marshalState) marshalStruct(s reflect.Value) {
	ms.WriteByte('d')
	fields := getExportedFields(s)
	sort.Slice(
		fields,
		func(i, j int) bool { return bencodeName(fields[i]) < bencodeName(fields[j]) },
	)
	for _, f := range fields {
		ms.marshalBytes([]byte(bencodeName(f)))
		ms.marshal(s.FieldByName(f.Name))
	}
	ms.WriteByte('e')
}

func getExportedFields(s reflect.Value) (fields []reflect.StructField) {
	for i := 0; i < s.NumField(); i++ {
		field := s.Type().Field(i)
		if field.Anonymous || field.PkgPath != "" {
			continue
		}
		if bencodeName(field) == "-" {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}

func bencodeName(field reflect.StructField) string {
	if tag, ok := field.Tag.Lookup("bencode"); ok {
		return tag
	}
	return field.Name
}

func handlePanic(err *error) {
	if r := recover(); r != nil {
		if _, ok := r.(runtime.Error); ok {
			panic(r)
		} else if _, ok := r.(error); ok {
			*err = r.(error)
		} else if _, ok := r.(string); ok {
			*err = errors.New(r.(string))
		} else {
			panic(r)
		}

	}
}
