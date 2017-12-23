package bencode

import (
	"reflect"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	var testCases = []struct {
		in  string
		out interface{}
	}{
		{"i123e", int64(123)},
		{"i-123e", int64(-123)},
		{"i0e", int64(0)},
		{"i9223372036854775807e", int64(9223372036854775807)},
		{"i-9223372036854775808e", int64(-9223372036854775808)},

		{"i18446744073709551615e", uint64(18446744073709551615)},

		{"4:cool", []byte("cool")},
		{"5:pizza", []byte("pizza")},

		{"li1ei2ei3ee", []interface{}{int64(1), int64(2), int64(3)}},
		{"li1e2:xde", []interface{}{int64(1), []byte("xd")}},

		{"d3:foo3:bar5:pizza4:coole",
			map[string]interface{}{"foo": []byte("bar"), "pizza": []byte("cool")}},
	}
	for _, tt := range testCases {
		var res interface{}
		err := Unmarshal([]byte(tt.in), &res)
		if err != nil {
			t.Fatalf("Error while unmarshalling %v: %v", tt.in, err)
		}
		if !reflect.DeepEqual(res, tt.out) {
			t.Fatalf("Unmarshal %v err: wanted %v(%t) got %v(%t)",
				tt.in, tt.out, tt.out, res, res)
		}
	}
}

func TestUnmarshalList2Slice(t *testing.T) {
	in := "li1ei2ei3ee"
	out := []int{1, 2, 3}
	var res []int

	err := Unmarshal([]byte(in), &res)
	if err != nil {
		t.Fatalf("Error while unmarshalling %v: %v", in, err)
	}
	if !reflect.DeepEqual(res, out) {
		t.Fatalf("Unmarshal %v err: wanted %v(%t) got %v(%t)",
			in, out, out, res, res)
	}
}

func TestUnmarshalList2Array(t *testing.T) {
	in := "li1ei2ei3ee"
	out := [5]int{1, 2, 3, 0, 0}
	var res [5]int

	err := Unmarshal([]byte(in), &res)
	if err != nil {
		t.Fatalf("Error while unmarshalling %v: %v", in, err)
	}
	if !reflect.DeepEqual(res, out) {
		t.Fatalf("Unmarshal %v err: wanted %v(%t) got %v(%t)",
			in, out, out, res, res)
	}
}

func TestUnmarshalDict2Struct(t *testing.T) {
	type TestStruct struct {
		Foo string
		Bar int
		baz int
	}
	type TestStruct2 struct {
		Ts TestStruct
		Hi string
	}

	testCases1 := []struct {
		in  string
		out TestStruct
	}{
		{"d3:Bari0e3:Foo3:bene", TestStruct{Foo: "ben"}},
		{"d3:Bari1e3:Foo3:bene", TestStruct{Foo: "ben", Bar: 1}},
		{"d3:Bari1e3:Foo3:ben3:bazi1ee", TestStruct{Foo: "ben", Bar: 1}},
		{"d3:Bari1e3:Foo3:ben5:Pizza4:coole", TestStruct{Foo: "ben", Bar: 1}},
	}
	for _, tt := range testCases1 {
		var res TestStruct
		err := Unmarshal([]byte(tt.in), &res)
		if err != nil {
			t.Fatalf("Error while unmarshalling %v: %v", tt.in, err)
		}
		if !reflect.DeepEqual(res, tt.out) {
			t.Fatalf("Unmarshal %v err: wanted %v(%t) got %v(%t)",
				tt.in, tt.out, tt.out, res, res)
		}
	}

	testCases2 := []struct {
		in  string
		out TestStruct2
	}{
		{
			"d2:Hi5:Hello2:Tsd3:Bari1e3:Foo3:benee",
			TestStruct2{
				Ts: TestStruct{Foo: "ben", Bar: 1},
				Hi: "Hello",
			},
		},
		{
			"d2:Hi5:Helloe",
			TestStruct2{
				Hi: "Hello",
			},
		},
	}
	for _, tt := range testCases2 {
		var res TestStruct2
		err := Unmarshal([]byte(tt.in), &res)
		if err != nil {
			t.Fatalf("Error while unmarshalling %v: %v", tt.in, err)
		}
		if !reflect.DeepEqual(res, tt.out) {
			t.Fatalf("Unmarshal %v err: wanted %v(%t) got %v(%t)",
				tt.in, tt.out, tt.out, res, res)
		}
	}

}

// TODO invalid test cases
