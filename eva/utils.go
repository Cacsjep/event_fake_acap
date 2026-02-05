package main

import (
	"math/rand"
	"strings"
)

func sanitizeEventName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "")
}

func RandomFloatInRange(start, end float64) float64 {
	return start + (end-start)*rand.Float64()
}

func RandomIntInRange(start, end int) int {
	return rand.Intn(end-start+1) + start
}

func RandomStringFromSlice(choices []string) string {
	if len(choices) == 0 {
		return ""
	}
	return choices[rand.Intn(len(choices))]
}

func RandomBool() bool {
	return rand.Intn(2) == 0
}
