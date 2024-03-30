package enviro

import (
	"golang.org/x/exp/constraints"
	"strconv"
	"strings"
)

func parseStringSlice(value string, trim bool) ([]string, error) {
	// Split the string by comma to get slice elements
	elements := strings.Split(value, ",")
	if trim {
		for i, elem := range elements {
			elements[i] = strings.TrimSpace(elem)
		}
	}
	return elements, nil
}

func parseIntSlice[T constraints.Signed](value string, bitSize int) ([]T, error) {
	elements := strings.Split(value, ",")
	intSlice := make([]T, 0, len(elements))
	for _, elem := range elements {
		i, err := strconv.ParseInt(strings.TrimSpace(elem), 10, bitSize)
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, T(i))
	}
	return intSlice, nil
}

func parseUintSlice[T constraints.Unsigned](value string, bitSize int) ([]T, error) {
	elements := strings.Split(value, ",")
	intSlice := make([]T, 0, len(elements))
	for _, elem := range elements {
		i, err := strconv.ParseUint(strings.TrimSpace(elem), 10, bitSize)
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, T(i))
	}
	return intSlice, nil
}

func parseFloatSlice[T constraints.Float](value string, bitSize int) ([]T, error) {
	elements := strings.Split(value, ",")
	intSlice := make([]T, 0, len(elements))
	for _, elem := range elements {
		i, err := strconv.ParseFloat(strings.TrimSpace(elem), bitSize)
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, T(i))
	}
	return intSlice, nil
}
