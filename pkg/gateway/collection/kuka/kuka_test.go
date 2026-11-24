package kuka

import "testing"

const sampleXML = `<Robot><Axis>A1</Axis><Position>12.34</Position><Status>OK</Status></Robot>`

func TestFindXPath(t *testing.T) {
	root, err := parseXML([]byte(sampleXML))
	if err != nil {
		t.Fatal(err)
	}
	text, err := findXPath(root, "//Robot/Position")
	if err != nil {
		t.Fatal(err)
	}
	if text != "12.34" {
		t.Fatalf("text = %q", text)
	}
	value, err := convertValue(text, "DOUBLE")
	if err != nil || value != 12.34 {
		t.Fatalf("value=%v err=%v", value, err)
	}
}

func TestConvertBool(t *testing.T) {
	value, err := convertValue("true", "BOOL")
	if err != nil || value != true {
		t.Fatalf("value=%v err=%v", value, err)
	}
}
