package bencode

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

func Unmarshal(data []byte, v interface{}) (err error) {
	defer handlePanic(&err)
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Ptr {
		panic(fmt.Errorf("cannot unmarshal into non-ptr type: %t", v))
	}

	us := unmarshalState{*bufio.NewReader(bytes.NewReader(data))}
	us.unmarshal(value)
	return
}

type unmarshalState struct {
	bufio.Reader
}

func (us *unmarshalState) unmarshal(v reflect.Value) {
	v = indirect(v)

	b := us.peekByte()
	switch {
	case b == 'l':
		us.skipByte()
		us.unmarshalList(v)
	case b == 'i':
		us.skipByte()
		us.unmarshalInt(v)
	case b == 'd':
		us.skipByte()
		us.unmarshalDict(v)
	case '0' <= b && b <= '9':
		us.unmarshalBytes(v)
	default:
		panic(fmt.Errorf("unexpected character %v", b))
	}
}

func (us *unmarshalState) unmarshalList(v reflect.Value) {
	if v.Kind() == reflect.Interface {
		v.Set(reflect.ValueOf([]interface{}{}))
	}

	var unmarshalElem func(i int)

	switch v.Kind() {
	case reflect.Interface:
		unmarshalElem = func(i int) {
			var elem interface{}
			us.unmarshal(reflect.ValueOf(&elem))
			v.Set(reflect.Append(v.Elem(), reflect.ValueOf(elem)))
		}

	case reflect.Array, reflect.Slice:
		unmarshalElem = func(i int) {
			elem := reflect.Indirect(reflect.New(v.Type().Elem()))
			us.unmarshal(elem)
			if i < v.Len() {
				v.Index(i).Set(elem)
			} else if v.Kind() == reflect.Slice {
				v.Set(reflect.Append(v, elem))
			}
		}

	default:
		panic(typeMismatch("list", v))
	}

	for i := 0; ; i++ {
		if us.peekByte() == 'e' {
			us.skipByte()
			break
		}
		unmarshalElem(i)
	}

}

func (us *unmarshalState) unmarshalInt(v reflect.Value) {
	data := us.readStringUntil('e')

	switch v.Kind() {
	case reflect.Interface:
		n, err := strconv.ParseInt(data, 10, 0)
		if err != nil && err.(*strconv.NumError).Err == strconv.ErrRange {
			un, err := strconv.ParseUint(data, 10, 0)
			if err != nil {
				panic(err)
			}
			v.Set(reflect.ValueOf(un))
			break
		} else if err != nil {
			panic(err)
		}
		v.Set(reflect.ValueOf(n))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(data, 10, 0)
		if err != nil {
			panic(err)
		}
		if v.OverflowInt(n) {
			panic(fmt.Errorf("%v overflows int", n))
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(data, 10, 0)
		if err != nil {
			panic(err)
		}
		if v.OverflowUint(n) {
			panic(fmt.Errorf("%v overflows uint", n))
		}
		v.SetUint(n)
	default:
		panic(typeMismatch("integer", v))
	}
}

func (us *unmarshalState) unmarshalDict(v reflect.Value) {
	switch v.Kind() {
	case reflect.Interface:
		us.unmarshalDict2Interface(v)
	case reflect.Map:
		us.unmarshalDict2Map(v)
	case reflect.Struct:
		us.unmarshalDict2Struct(v)
	default:
		panic(typeMismatch("dict", v))
	}
}

func (us *unmarshalState) unmarshalDict2Interface(v reflect.Value) {
	m := make(map[string]interface{})
	v.Set(reflect.ValueOf(m))
	us.unmarshalDict2Map(v.Elem())
}

func (us *unmarshalState) unmarshalDict2Map(v reflect.Value) {
	if v.Type().Key().Kind() != reflect.String {
		panic(errors.New("map keys must be of type string"))
	}

	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}

	for {
		if us.peekByte() == 'e' {
			us.skipByte()
			break
		}

		var key string
		us.unmarshal(reflect.ValueOf(&key))

		val := reflect.Indirect(reflect.New(v.Type().Elem()))
		us.unmarshal(val)

		v.SetMapIndex(reflect.ValueOf(key), val)
	}
}

func (us *unmarshalState) unmarshalDict2Struct(v reflect.Value) {
	fields := structFields(v)
	for {
		if us.peekByte() == 'e' {
			us.skipByte()
			break
		}

		var key string
		us.unmarshal(reflect.ValueOf(&key))

		if field, ok := fields[key]; ok {
			us.unmarshal(field)
		} else {
			var dummy interface{}
			us.unmarshal(reflect.ValueOf(&dummy))
			dummy = nil
		}
	}
}

func (us *unmarshalState) unmarshalBytes(v reflect.Value) {
	lenStr := us.readStringUntil(':')
	length, err := strconv.ParseUint(lenStr, 10, 0)
	if err != nil {
		panic(err)
	}

	bytes := make([]byte, length)
	_, err = io.ReadFull(us, bytes)
	if err != nil {
		panic(err)
	}

	switch v.Kind() {
	case reflect.Interface:
		v.Set(reflect.ValueOf(bytes))
	case reflect.String:
		v.SetString(string(bytes))
	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.Uint8 {
			panic(typeMismatch("bytes", v))
		}
		v.Set(reflect.ValueOf(bytes))
	default:
		panic(typeMismatch("bytes", v))
	}
}

func (us *unmarshalState) readStringUntil(delim byte) string {
	data, err := us.ReadString(delim)
	if err != nil {
		panic(err)
	}
	return data[:len(data)-1]
}

func (us *unmarshalState) peekByte() byte {
	b, err := us.ReadByte()
	if err != nil {
		panic(err)
	}
	err = us.UnreadByte()
	if err != nil {
		panic(err)
	}
	return b
}

func (us *unmarshalState) skipByte() {
	_, err := us.ReadByte()
	if err != nil {
		panic(err)
	}
}

func typeMismatch(bencodeType string, v reflect.Value) error {
	return fmt.Errorf("cannot unmarshal %s into %v", bencodeType, v.Kind())
}

func structFields(s reflect.Value) map[string]reflect.Value {
	fields := make(map[string]reflect.Value)
	for i := 0; i < s.NumField(); i++ {
		fieldValue := s.Field(i)
		if !fieldValue.CanSet() {
			continue
		}
		structField := s.Type().Field(i)
		fields[structField.Name] = fieldValue
		if tag, ok := structField.Tag.Lookup("bencode"); ok {
			fields[tag] = fieldValue
		}
	}
	return fields
}

func indirect(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {
		if v.Kind() != reflect.Ptr {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}
