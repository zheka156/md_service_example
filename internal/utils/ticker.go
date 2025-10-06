package utils

import (
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

func ValidateTicker(ticker string) bool {
	var valid = regexp.MustCompile((`^[A-Z]{1,5}$`))
	return valid.MatchString(ticker)
}

func RemoveDuplicates(input []string) []string {
	inputMap := make(map[string]bool)
	output := []string{}
	for _, value := range input {
		if _, ok := inputMap[value]; !ok {
			inputMap[value] = true
			output = append(output, value)
		}
	}
	return output
}

func StripUSDT(ticker string) string {
	return ticker[:len(ticker)-4]
}

func CreateCoinRegexp(coins []string) (*regexp.Regexp, error) {
	pattern := `(?i)\b(` + strings.Join(coins, "|") + `)\b`

	// Compile the pattern into a regexp.Regexp object
	return regexp.Compile(pattern)
}

func CreateCoinQuantityRegexp(coins []string) (*regexp.Regexp, error) {
	pattern := `(?i)\b(` + strings.Join(coins, "|") + `),\s*\d+(\.\d+)?\b`
	return regexp.Compile(pattern)
}

func GetTickersFromUserInput(input string) (output []string) {
	noSpaceStr := strings.ReplaceAll(input, " ", "")
	inputList := strings.Split(noSpaceStr, ",")
	for _, val := range inputList {
		if val != "" {
			output = append(output, strings.ToUpper(val))
		}
	}
	return output
}

func MapTickersToQuantityFromUserInput(input string) (result map[string]decimal.Decimal) {
	noSpaceStr := strings.ReplaceAll(input, " ", "")
	inputList := strings.Split(noSpaceStr, ",")
	var singleInput []string
	for _, val := range inputList {
		if val != "" {
			singleInput = append(singleInput, strings.ToUpper(val))
		}
	}
	result = make(map[string]decimal.Decimal)
	result[singleInput[0]] = decimal.RequireFromString(singleInput[1])
	return result
}

func CreateRemoveRegexp() (*regexp.Regexp, error) {
	return regexp.Compile(`(?i)\bremove\b\s*\b([A-Z]{1,5})\b`)
}