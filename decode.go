package bencode

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strconv"
)

func Unmarshal(data []byte, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Ptr {
		panic(fmt.Errorf("can't unmarshal into non-ptr type: %t", v))
	}

	us := unmarshalState{*bufio.NewReader(bytes.NewReader(data))}
	us.unmarshal(value)
	return
}

type unmarshalState struct {
	bufio.Reader
}

func (us *unmarshalState) unmarshal(v reflect.Value) {
	// v indirect
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

	b, err := us.ReadByte()
	if err != nil {
		panic(err)
	}
	switch {
	case b == 'l':
		us.unmarshalList(v)
	case b == 'i':
		us.unmarshalInt(v)
	case b == 'd':
		us.unmarshalDict(v)
	case '0' <= b && b <= '9':
		us.UnreadByte()
		us.unmarshalBytes(v)
	default:
		panic(fmt.Errorf("unexpected character %v", b))
	}
}

func (us *unmarshalState) unmarshalList(v reflect.Value) {
	if v.Kind() == reflect.Interface {
		v.Set(reflect.ValueOf([]interface{}{}))
	}

	for i := 0; ; i++ {
		c, err := us.ReadByte()
		if err != nil {
			panic(err)
		}
		if c == 'e' {
			break
		}
		err = us.UnreadByte()
		if err != nil {
			panic(err)
		}

		switch v.Kind() {
		case reflect.Interface:
			var elem interface{}
			us.unmarshal(reflect.ValueOf(&elem))
			v.Set(reflect.Append(v.Elem(), reflect.ValueOf(elem)))
		case reflect.Array:
			// TODO indirect not needed?
			elem := reflect.Indirect(reflect.New(v.Type().Elem()))
			us.unmarshal(elem)
			if i < v.Len() {
				v.Index(i).Set(elem)
			}
		case reflect.Slice:
			elem := reflect.Indirect(reflect.New(v.Type().Elem()))
			us.unmarshal(elem)
			if i < v.Len() {
				v.Index(i).Set(elem)
			} else {
				v.Set(reflect.Append(v, elem))
			}
		default:
			panic(typeMismatch("list", v))
		}
	}

}

func (us *unmarshalState) unmarshalInt(v reflect.Value) {
	data, err := us.readStringUntil('e')
	if err != nil {
		panic(err)
	}

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
		fmt.Print("unmarshaling struct")
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
		c, err := us.ReadByte()
		if err != nil {
			panic(err)
		}
		if c == 'e' {
			break
		}
		err = us.UnreadByte()
		if err != nil {
			panic(err)
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
		c, err := us.ReadByte()
		if err != nil {
			panic(err)
		}
		if c == 'e' {
			break
		}
		err = us.UnreadByte()
		if err != nil {
			panic(err)
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
	lenStr, err := us.readStringUntil(':')
	if err != nil {
		panic(err)
	}
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

func (us *unmarshalState) readStringUntil(delim byte) (data string, err error) {
	data, err = us.ReadString(delim)
	if err != nil {
		return
	}
	data = data[:len(data)-1]
	return
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
