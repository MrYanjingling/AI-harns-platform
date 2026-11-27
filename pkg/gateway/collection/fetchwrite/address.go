package fetchwrite

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	OrgDB   byte = 1 // 16-bit word addressed
	OrgM    byte = 2 // 8-bit byte addressed
	OrgI    byte = 3 // 8-bit byte addressed
	OrgQ    byte = 4 // 8-bit byte addressed
	OrgPEPA byte = 5 // 8-bit byte addressed
	OrgZ    byte = 6 // 16-bit word addressed
	OrgT    byte = 7 // 16-bit word addressed

	maxStartOffset = 0x7FF
	maxDBNumber    = 0xFF
)

// Address represents a parsed Siemens S5 Fetch/Write memory address.
type Address struct {
	Org        byte
	DB         byte
	Start      uint16 // unit offset: word index for DB/Z/T, byte index for M/I/Q/PEPA
	UnitCount  uint16 // number of units (words for DB/Z/T, bytes for M/I/Q/PEPA)
	ByteOffset uint8  // sub-byte offset inside the first word (0 or 1) for DBB/DBX
	Bit        uint8  // bit index (0..15 for word bit, 0..7 for byte bit)
	ByteLength uint16 // logical value byte length
	IsBit      bool
	WordBit    bool // true when Bit is 0..15 across a 16-bit big-endian word (e.g. DB1.DBW0.7)
}

func unitBytes(org byte) int {
	switch org {
	case OrgDB, OrgZ, OrgT:
		return 2
	default:
		return 1
	}
}

// ParseAddress parses S5 Fetch/Write addresses:
//   - DB<n>.DBW<word>[.<bit|len>]
//   - DB<n>.DBD<word>
//   - DB<n>.DBB<byte>[.<len>]
//   - DB<n>.DBX<byte>.<bit>
//   - M<byte>[.<bit|len>], I<byte>[.<bit|len>], Q<byte>[.<bit|len>], PEPA<byte>[.<bit|len>]
//   - Z<word>, T<word>
func ParseAddress(raw string, dataType string) (Address, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return Address{}, fmt.Errorf("fetch/write address is empty")
	}
	upper := strings.ToUpper(value)
	dt := strings.ToLower(strings.TrimSpace(dataType))

	var addr Address
	var suffix uint16
	var hasSuffix bool

	switch {
	case strings.HasPrefix(upper, "DB"):
		var err error
		addr, suffix, hasSuffix, err = parseDBAddress(upper, dt)
		if err != nil {
			return Address{}, err
		}
	case strings.HasPrefix(upper, "PEPA"):
		var err error
		addr, suffix, hasSuffix, err = parseByteAreaAddress(OrgPEPA, upper[4:], dt)
		if err != nil {
			return Address{}, err
		}
	case strings.HasPrefix(upper, "M"):
		var err error
		addr, suffix, hasSuffix, err = parseByteAreaAddress(OrgM, upper[1:], dt)
		if err != nil {
			return Address{}, err
		}
	case strings.HasPrefix(upper, "I"):
		var err error
		addr, suffix, hasSuffix, err = parseByteAreaAddress(OrgI, upper[1:], dt)
		if err != nil {
			return Address{}, err
		}
	case strings.HasPrefix(upper, "Q"):
		var err error
		addr, suffix, hasSuffix, err = parseByteAreaAddress(OrgQ, upper[1:], dt)
		if err != nil {
			return Address{}, err
		}
	case strings.HasPrefix(upper, "Z"):
		var err error
		addr, suffix, hasSuffix, err = parseWordAreaAddress(OrgZ, upper[1:], dt)
		if err != nil {
			return Address{}, err
		}
	case strings.HasPrefix(upper, "T"):
		var err error
		addr, suffix, hasSuffix, err = parseWordAreaAddress(OrgT, upper[1:], dt)
		if err != nil {
			return Address{}, err
		}
	default:
		return Address{}, fmt.Errorf("unsupported fetch/write address %q", raw)
	}

	if err := finalizeAddress(&addr, dt, suffix, hasSuffix); err != nil {
		return Address{}, err
	}
	return addr, nil
}

func parseDBAddress(upper string, dt string) (Address, uint16, bool, error) {
	dot := strings.IndexByte(upper, '.')
	if dot < 3 {
		return Address{}, 0, false, fmt.Errorf("invalid fetch/write DB address %q", upper)
	}
	dbNum, err := strconv.ParseUint(upper[2:dot], 10, 16)
	if err != nil || dbNum > maxDBNumber {
		return Address{}, 0, false, fmt.Errorf("invalid fetch/write DB number in %q", upper)
	}
	rest := upper[dot+1:]
	addr := Address{Org: OrgDB, DB: byte(dbNum)}

	switch {
	case strings.HasPrefix(rest, "DBW"):
		num, suf, hasSuf, err := splitOffsetSuffix(rest[3:])
		if err != nil || num > maxStartOffset {
			return Address{}, 0, false, fmt.Errorf("invalid fetch/write DBW address %q", upper)
		}
		addr.Start = num
		if dt == "bool" || dt == "bit" {
			addr.IsBit = true
			addr.WordBit = true
			if hasSuf {
				if suf > 15 {
					return Address{}, 0, false, fmt.Errorf("fetch/write word bit out of range in %q", upper)
				}
				addr.Bit = uint8(suf)
			}
		}
		return addr, suf, hasSuf, nil

	case strings.HasPrefix(rest, "DBD"):
		num, suf, hasSuf, err := splitOffsetSuffix(rest[3:])
		if err != nil || num > maxStartOffset {
			return Address{}, 0, false, fmt.Errorf("invalid fetch/write DBD address %q", upper)
		}
		addr.Start = num
		return addr, suf, hasSuf, nil

	case strings.HasPrefix(rest, "DBB"):
		num, suf, hasSuf, err := splitOffsetSuffix(rest[3:])
		if err != nil || num/2 > maxStartOffset {
			return Address{}, 0, false, fmt.Errorf("invalid fetch/write DBB address %q", upper)
		}
		addr.Start = num / 2
		addr.ByteOffset = uint8(num % 2)
		if dt == "bool" || dt == "bit" {
			addr.IsBit = true
			if hasSuf {
				if suf > 7 {
					return Address{}, 0, false, fmt.Errorf("fetch/write byte bit out of range in %q", upper)
				}
				addr.Bit = uint8(suf)
			}
		}
		return addr, suf, hasSuf, nil

	case strings.HasPrefix(rest, "DBX"):
		num, suf, hasSuf, err := splitOffsetSuffix(rest[3:])
		if err != nil || !hasSuf || suf > 7 || num/2 > maxStartOffset {
			return Address{}, 0, false, fmt.Errorf("invalid fetch/write DBX address %q", upper)
		}
		addr.Start = num / 2
		addr.ByteOffset = uint8(num % 2)
		addr.Bit = uint8(suf)
		addr.IsBit = true
		return addr, suf, hasSuf, nil

	default:
		return Address{}, 0, false, fmt.Errorf("unsupported fetch/write DB format %q", upper)
	}
}

func parseByteAreaAddress(org byte, rest string, dt string) (Address, uint16, bool, error) {
	num, suf, hasSuf, err := splitOffsetSuffix(rest)
	if err != nil || num > maxStartOffset {
		return Address{}, 0, false, fmt.Errorf("invalid fetch/write byte area address %q", rest)
	}
	addr := Address{Org: org, Start: num}
	if dt == "bool" || dt == "bit" {
		addr.IsBit = true
		if hasSuf {
			if suf > 7 {
				return Address{}, 0, false, fmt.Errorf("fetch/write bit out of range in %q", rest)
			}
			addr.Bit = uint8(suf)
		}
	}
	return addr, suf, hasSuf, nil
}

func parseWordAreaAddress(org byte, rest string, dt string) (Address, uint16, bool, error) {
	num, suf, hasSuf, err := splitOffsetSuffix(rest)
	if err != nil || num > maxStartOffset {
		return Address{}, 0, false, fmt.Errorf("invalid fetch/write word area address %q", rest)
	}
	addr := Address{Org: org, Start: num}
	if dt == "bool" || dt == "bit" {
		addr.IsBit = true
		addr.WordBit = true
		if hasSuf {
			if suf > 15 {
				return Address{}, 0, false, fmt.Errorf("fetch/write word bit out of range in %q", rest)
			}
			addr.Bit = uint8(suf)
		}
	}
	return addr, suf, hasSuf, nil
}

func splitOffsetSuffix(rest string) (uint16, uint16, bool, error) {
	if rest == "" {
		return 0, 0, false, fmt.Errorf("empty offset")
	}
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		num, err := strconv.ParseUint(rest[:dot], 10, 16)
		if err != nil {
			return 0, 0, false, err
		}
		suf, err := strconv.ParseUint(rest[dot+1:], 10, 16)
		if err != nil {
			return 0, 0, false, err
		}
		return uint16(num), uint16(suf), true, nil
	}
	num, err := strconv.ParseUint(rest, 10, 16)
	if err != nil {
		return 0, 0, false, err
	}
	return uint16(num), 0, false, nil
}

func finalizeAddress(addr *Address, dt string, suffix uint16, hasSuffix bool) error {
	var byteLen uint16
	switch dt {
	case "bool", "bit":
		if addr.WordBit {
			byteLen = 2
		} else {
			byteLen = 1
		}
		addr.IsBit = true
	case "int16", "uint16":
		byteLen = 2
	case "int32", "uint32", "float32":
		byteLen = 4
	case "int64", "uint64", "float64":
		byteLen = 8
	case "string", "byte_array":
		if !hasSuffix || suffix == 0 {
			return fmt.Errorf("fetch/write %s requires a positive length suffix", dt)
		}
		byteLen = suffix
	default:
		return fmt.Errorf("unsupported fetch/write data type %q", dt)
	}
	addr.ByteLength = byteLen
	totalBytes := int(addr.ByteOffset) + int(byteLen)
	ub := unitBytes(addr.Org)
	addr.UnitCount = uint16((totalBytes + ub - 1) / ub)
	if addr.UnitCount == 0 {
		addr.UnitCount = 1
	}
	if int(addr.Start)+int(addr.UnitCount) > 0x1000 {
		return fmt.Errorf("fetch/write address range exceeds limit")
	}
	return nil
}
