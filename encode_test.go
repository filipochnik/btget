package bencode

import (
	"bytes"
	"testing"
)

var marshalTestCases = []struct {
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
}

func TestMarshal(t *testing.T) {
	for _, tt := range marshalTestCases {
		b, err := Marshal(tt.in)
		if err != nil {
			t.Fatalf("Error while marshalling %v: %v", tt.in, err)
		}
		if bytes.Equal(b, []byte(tt.out)) != true {
			t.Fatalf("Marshal int %v err: wanted %v got %v", tt.in, tt.out, string(b))
		}
	}
}

// TODO invalid test cases
// TODO property cheking
