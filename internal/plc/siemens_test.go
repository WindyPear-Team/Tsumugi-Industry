package plc

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestParseS7Address(t *testing.T) {
	cases := []struct {
		input  string
		area   byte
		db     int
		offset int
		bit    int
	}{
		{input: "DB1.DBX0.3", area: 'D', db: 1, offset: 0, bit: 3},
		{input: "DB12.DBD4", area: 'D', db: 12, offset: 4, bit: -1},
		{input: "M10.7", area: 'M', offset: 10, bit: 7},
		{input: "IB2", area: 'I', offset: 2, bit: -1},
		{input: "QW4", area: 'Q', offset: 4, bit: -1},
	}
	for _, testCase := range cases {
		address, err := parseS7Address(testCase.input)
		if err != nil {
			t.Fatalf("parse %s: %v", testCase.input, err)
		}
		if address.area != testCase.area || address.db != testCase.db || address.offset != testCase.offset || address.bit != testCase.bit {
			t.Errorf("parse %s = %+v", testCase.input, address)
		}
	}
}

func TestEncodeS7Value(t *testing.T) {
	word, err := encodeS7Value(float64(258), "UINT")
	if err != nil || len(word) != 2 || binary.BigEndian.Uint16(word) != 258 {
		t.Fatalf("UINT encoding = %v, %v", word, err)
	}
	real, err := encodeS7Value(1.5, "REAL")
	if err != nil || len(real) != 4 || math.Float32frombits(binary.BigEndian.Uint32(real)) != 1.5 {
		t.Fatalf("REAL encoding = %v, %v", real, err)
	}
	if value, err := encodeS7Value(false, "BOOL"); err != nil || len(value) != 1 || value[0] != 0 {
		t.Fatalf("BOOL encoding = %v, %v", value, err)
	}
	if _, err := encodeS7Value(float64(256), "BYTE"); err == nil {
		t.Fatal("expected BYTE range error")
	}
}
