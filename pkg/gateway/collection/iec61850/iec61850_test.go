package iec61850

import (
	"testing"

	iec "github.com/otfabric/go-iec61850"
	"github.com/otfabric/go-mms"
)

func TestParseAddress(t *testing.T) {
	address, err := ParseAddress("LD0/LLN0.Health", "ST")
	if err != nil {
		t.Fatal(err)
	}
	if string(address.Reference.FC) != "ST" || address.Reference.LD != "LD0" || address.Reference.LN != "LLN0" {
		t.Fatalf("address=%+v", address.Reference)
	}
	address, err = ParseAddress("LD0/GGIO1.AnIn1.mag.f[MX]", "ST")
	if err != nil {
		t.Fatal(err)
	}
	if string(address.Reference.FC) != "MX" {
		t.Fatalf("fc=%s", address.Reference.FC)
	}
	for _, value := range []string{"", "LD0", "LD0/", "LD0/LLN0.Health[BAD]"} {
		if _, err := ParseAddress(value, "ST"); err == nil {
			t.Fatalf("%q should fail", value)
		}
	}
}

func TestValueCodec(t *testing.T) {
	value, err := convertValue(iec.NewValue(mms.NewFloat(1.5)))
	if err != nil {
		t.Fatal(err)
	}
	if value != 1.5 {
		t.Fatalf("value=%#v", value)
	}
	encoded, err := encodeValue(int32(42), "INT32")
	if err != nil {
		t.Fatal(err)
	}
	if encoded == nil {
		t.Fatal("encoded value is nil")
	}
}
