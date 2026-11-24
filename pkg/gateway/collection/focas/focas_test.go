package focas

import "testing"

func TestParseAddress(t *testing.T) {
	address, err := ParseAddress("R100", "UINT16")
	if err != nil || address.Kind != "pmc" || address.Area != 'R' || address.Number != 100 {
		t.Fatalf("pmc = %+v err=%v", address, err)
	}
	address, err = ParseAddress("X0.3", "BIT")
	if err != nil || address.Bit != 3 {
		t.Fatalf("bit = %+v err=%v", address, err)
	}
	address, err = ParseAddress("absolute.X", "DOUBLE")
	if err != nil || address.Axis != 0 || address.Kind != "absolute" {
		t.Fatalf("axis = %+v err=%v", address, err)
	}
	address, err = ParseAddress("runState", "UINT16")
	if err != nil || address.Kind != "runstate" {
		t.Fatalf("state = %+v err=%v", address, err)
	}
}

func TestLibraryCandidates(t *testing.T) {
	if lens := len(libraryCandidates); lens != 4 {
		t.Fatalf("candidate count = %d", lens)
	}
}
