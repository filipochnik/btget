package bencode

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarshal(t *testing.T) {
	type TestStruct struct {
		Foo string
		Bar int
		baz int
	}
	type TestStruct2 struct {
		Ts TestStruct
		Hi string
	}
	var testCases = []struct {
		in  interface{}
		out string
	}{
		{int(123), "i123e"},
		{int(-123), "i-123e"},
		{int(0), "i0e"},
		{int(9223372036854775807), "i9223372036854775807e"},
		{int(-9223372036854775808), "i-9223372036854775808e"},

		{uint(123), "i123e"},
		{uint(0), "i0e"},
		{uint(18446744073709551615), "i18446744073709551615e"},

		{[]int{}, "le"},
		{[]int{1}, "li1ee"},
		{[]int{1, 3, 3, 7}, "li1ei3ei3ei7ee"},
		{[3]int{997, 1916, -1410}, "li997ei1916ei-1410ee"},
		{[]interface{}{7}, "li7ee"},
		{[]interface{}{7, "lol", []int{4, 2}}, "li7e3:lolli4ei2eee"},

		{"", "0:"},
		{"foo/bar", "7:foo/bar"},
		{[]byte("$BenCode$"), "9:$BenCode$"},

		{map[string]int{"foo": 1}, "d3:fooi1ee"},
		{map[string]int{"foo": 1, "bar": 2}, "d3:bari2e3:fooi1ee"},
		{map[string]interface{}{"foo": "bar", "baz": []int{2}}, "d3:bazli2ee3:foo3:bare"},

		{TestStruct{Foo: "ben"}, "d3:Bari0e3:Foo3:bene"},
		{TestStruct{Foo: "ben", Bar: 1, baz: 2}, "d3:Bari1e3:Foo3:bene"},
		{
			TestStruct2{
				Ts: TestStruct{Foo: "ben", Bar: 1, baz: 2},
				Hi: "Hello",
			},
			"d2:Hi5:Hello2:Tsd3:Bari1e3:Foo3:benee",
		},
	}

	for _, tc := range testCases {
		b, err := Marshal(tc.in)
		if err != nil {
			t.Fatalf("Error while marshalling %v: %v", tc.in, err)
		}
		if bytes.Equal(b, []byte(tc.out)) != true {
			t.Fatalf("Marshal %v err: wanted %v got %v", tc.in, tc.out, string(b))
		}
	}
}

func TestMarshalInvalid(t *testing.T) {
	var testCases = []struct {
		in          interface{}
		errContains string
	}{
		{[]chan int{make(chan int)}, "unsupported type"},
		{map[int]int{1: 1}, "cannot unmarshal map"},
	}
	for _, tc := range testCases {
		_, err := Marshal(tc.in)
		assertErrContains(t, err, tc.errContains)
	}

}

func assertErrContains(t *testing.T, err error, contains string) {
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), contains) {
		t.Fatalf("expected error containing \"%s\", got \"%s\" instead", contains, err.Error())
	}
}

// TODO property checking
