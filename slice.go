package enviro

import (
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

func parseIntSlice(value string) ([]int, error) {
	elements := strings.Split(value, ",")
	intSlice := make([]int, 0, len(elements))
	for _, elem := range elements {
		i, err := strconv.Atoi(strings.TrimSpace(elem))
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, i)
	}
	return intSlice, nil
}
