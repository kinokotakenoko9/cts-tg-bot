package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func stringToCompartmentNumber(s string) ([]int, bool) {
	parts := strings.Fields(s)
	if len(parts) == 0 || len(parts) > 9 {
		return nil, false
	}

	seen := make(map[int]bool)
	ints := make([]int, len(parts))

	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 9 || seen[n] {
			return nil, false
		}
		seen[n] = true
		ints[i] = n
	}
	return ints, true
}
func compartmentNumberToString(compartmentNumber []int) string {
	s := []string{}
	for _, n := range compartmentNumber {
		s = append(s, strconv.Itoa(n))
	}
	return strings.Join(s, " ")
}

func remove[T comparable](l []T, item T) []T {
	out := make([]T, 0)
	for _, element := range l {
		if element != item {
			out = append(out, element)
		}
	}
	return out
}

func loadCities() error {
	data, err := os.ReadFile("cities.json")
	if err != nil {
		log.Println("Error: reading file: ", err)
		return err
	}

	err = json.Unmarshal(data, &cities)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}

	return nil
}

func getCitiesWithPrefix(prefix string) []string {
	var result []string
	lowerPrefix := strings.ToLower(prefix)
	for city, _ := range cities {
		if strings.HasPrefix(strings.ToLower(city), lowerPrefix) {
			result = append(result, city)
		}
	}
	return result
}
