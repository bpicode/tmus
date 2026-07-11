package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ByteSize stores a byte count and marshals to TOML as a unit-bearing string.
type ByteSize int64

var byteSizeUnits = map[string]int64{
	"B":   1,
	"KB":  1000,
	"MB":  1000 * 1000,
	"GB":  1000 * 1000 * 1000,
	"TB":  1000 * 1000 * 1000 * 1000,
	"KiB": 1024,
	"MiB": 1024 * 1024,
	"GiB": 1024 * 1024 * 1024,
	"TiB": 1024 * 1024 * 1024 * 1024,
}

var byteSizeFormatUnits = []struct {
	unit string
	mult int64
}{
	{"TiB", byteSizeUnits["TiB"]},
	{"TB", byteSizeUnits["TB"]},
	{"GiB", byteSizeUnits["GiB"]},
	{"GB", byteSizeUnits["GB"]},
	{"MiB", byteSizeUnits["MiB"]},
	{"MB", byteSizeUnits["MB"]},
	{"KiB", byteSizeUnits["KiB"]},
	{"KB", byteSizeUnits["KB"]},
}

func (b ByteSize) String() string {
	value := int64(b)
	prefix := ""
	abs := uint64(value)
	if value < 0 {
		prefix = "-"
		abs = uint64(-(value + 1)) + 1
	}
	for _, unit := range byteSizeFormatUnits {
		mult := uint64(unit.mult)
		if abs >= mult && abs%mult == 0 {
			return fmt.Sprintf("%s%d%s", prefix, abs/mult, unit.unit)
		}
	}
	return fmt.Sprintf("%s%dB", prefix, abs)
}

// MarshalText serializes a byte size as an explicit unit-bearing string.
func (b ByteSize) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

// UnmarshalText parses a byte size string with an explicit supported unit.
func (b *ByteSize) UnmarshalText(text []byte) error {
	value, err := parseByteSize(string(text))
	if err != nil {
		return err
	}
	*b = ByteSize(value)
	return nil
}

func parseByteSize(value string) (int64, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return 0, fmt.Errorf("byte size is empty")
	}

	sign := int64(1)
	if raw[0] == '+' || raw[0] == '-' {
		if raw[0] == '-' {
			sign = -1
		}
		raw = raw[1:]
	}

	split := 0
	for split < len(raw) && raw[split] >= '0' && raw[split] <= '9' {
		split++
	}
	if split == 0 {
		return 0, fmt.Errorf("byte size must start with a number")
	}

	number, unit := raw[:split], raw[split:]
	mult, ok := byteSizeUnits[unit]
	if !ok {
		return 0, fmt.Errorf("unsupported byte size unit %q", unit)
	}

	magnitude, err := strconv.ParseUint(number, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse byte size: %w", err)
	}
	if magnitude > uint64(math.MaxInt64)/uint64(mult) {
		return 0, fmt.Errorf("byte size overflows int64")
	}

	bytes := int64(magnitude) * mult
	return sign * bytes, nil
}
