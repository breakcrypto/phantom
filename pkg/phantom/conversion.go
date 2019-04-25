package phantom

import (
	"strconv"
	"strings"
)

func ConvertVersionStringToInt(str string) uint32 {
	version := 0
	parts := strings.Split(str, ".")
	for _, part := range parts {
		version <<= 8
		value, _ := strconv.Atoi(part)
		version |= value
	}
	return uint32(version)
}