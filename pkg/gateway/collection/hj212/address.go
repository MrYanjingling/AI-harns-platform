package hj212

import (
	"fmt"
	"strings"
)

type Address struct {
	Command int
	Key     string
	Float   bool
}

var (
	rtFloatSuffixes = []string{"-Rtd", "-ZsRtd"}
	rtTextSuffixes  = []string{"-Flag", "-SampleTime", "-EFlag"}
	periodSuffixes  = []string{"-Cou", "-Min", "-Avg", "-Max", "-ZsMin", "-ZsAvg", "-ZsMax"}
)

func ParseAddress(value, dataType string) (Address, error) {
	value = strings.TrimSpace(value)
	index := strings.Index(value, "!")
	if index < 0 {
		return Address{}, fmt.Errorf("HJ212 address must start with RT!/MIN!/HOUR!/DAY!")
	}
	prefix := strings.ToUpper(strings.TrimSpace(value[:index]))
	key := strings.TrimSpace(value[index+1:])
	if key == "" {
		return Address{}, fmt.Errorf("HJ212 pollutant key is empty")
	}
	command := 0
	switch prefix {
	case "RT":
		command = 2011
	case "MIN":
		command = 2051
	case "HOUR":
		command = 2061
	case "DAY":
		command = 2031
	default:
		return Address{}, fmt.Errorf("unsupported HJ212 prefix %q", prefix)
	}
	float := false
	switch prefix {
	case "RT":
		if !matchSuffix(key, rtTextSuffixes) {
			if !matchSuffix(key, rtFloatSuffixes) {
				return Address{}, fmt.Errorf("unsupported HJ212 RT suffix")
			}
			float = true
		}
	default:
		if !strings.HasSuffix(key, "-Flag") {
			if !matchSuffix(key, periodSuffixes) {
				return Address{}, fmt.Errorf("unsupported HJ212 %s suffix", prefix)
			}
			float = true
		}
	}
	if float && !strings.EqualFold(dataType, "double") && !strings.EqualFold(dataType, "float") {
		return Address{}, fmt.Errorf("HJ212 numeric suffix requires DOUBLE")
	}
	if !float && !strings.EqualFold(dataType, "string") {
		return Address{}, fmt.Errorf("HJ212 string suffix requires STRING")
	}
	return Address{Command: command, Key: key, Float: float}, nil
}

func matchSuffix(value string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) && len(value) > len(suffix) {
			return true
		}
	}
	return false
}
