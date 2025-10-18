package utils

import (
	"strconv"
	"strings"
)

// FormatInt64WithCommas returns an int64 formatted with thousands separators.
func FormatInt64WithCommas(n int64) string {
	negative := n < 0
	if negative {
		n = -n
	}

	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}

	var builder strings.Builder
	if negative {
		builder.WriteByte('-')
	}

	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	builder.WriteString(s[:rem])

	for i := rem; i < len(s); i += 3 {
		builder.WriteByte(',')
		builder.WriteString(s[i : i+3])
	}

	return builder.String()
}

// FormatIntWithCommas returns an int formatted with thousands separators.
func FormatIntWithCommas(n int) string {
	return FormatInt64WithCommas(int64(n))
}
