package utility

import (
	"log"
	"strconv"
)

// ParseOfferPrice converts a string to float64 for use as a gRPC double.
// If parsing fails, it returns 0.0 as the default value.
func ParseStringToDouble(input string) float64 {
	if input == "" {
		return 0.0
	}

	value, err := strconv.ParseFloat(input, 64)
	if err != nil {
		log.Printf("invalid imput '%s': %v, defaulting to 0.0", input, err)
		return 0.0
	}

	return value
}
