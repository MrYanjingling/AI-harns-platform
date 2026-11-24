package ab5000

import "testing"

func TestParseAddressAndReadRequest(t *testing.T) {
	address, err := parseAddress("MyArray[2].Member", "INT32")
	if err != nil {
		t.Fatal(err)
	}
	request, err := readRequest(address)
	if err != nil {
		t.Fatal(err)
	}
	if request[0] != 0x4C {
		t.Fatalf("service = 0x%02X", request[0])
	}
}

func TestDataTypeCode(t *testing.T) {
	if code, err := cipTypeCode("SINT"); err != nil || code != 0xC2 {
		t.Fatalf("SINT code = 0x%X err=%v", code, err)
	}
	if code, err := cipTypeCode("REAL"); err != nil || code != 0xCA {
		t.Fatalf("REAL code = 0x%X err=%v", code, err)
	}
}
