package validator

import (
	"time"
)

var validGroupCodes = map[string]bool{
	"A": true, "B": true, "C": true, "D": true,
	"E": true, "F": true, "G": true, "H": true,
	"I": true, "J": true, "K": true, "L": true,
}

var validStageCodes = map[string]bool{
	"group_stage": true, "round_of_16": true,
	"quarter_finals": true, "semi_finals": true,
	"third_place": true, "final": true,
}

var validStatuses = map[string]bool{
	"scheduled": true, "finished": true,
}

var validFifaCodes = map[string]bool{
	// Group A
	"MEX": true, "RSA": true, "KOR": true, "CZE": true,
	// Group B
	"CAN": true, "BIH": true, "QAT": true, "SUI": true,
	// Group C
	"BRA": true, "MAR": true, "HAI": true, "SCO": true,
	// Group D
	"USA": true, "PAR": true, "AUS": true, "TUR": true,
	// Group E
	"GER": true, "CUW": true, "CIV": true, "ECU": true,
	// Group F
	"NED": true, "JPN": true, "SWE": true, "TUN": true,
	// Group G
	"BEL": true, "EGY": true, "IRN": true, "NZL": true,
	// Group H
	"ESP": true, "CPV": true, "KSA": true, "URU": true,
	// Group I
	"FRA": true, "SEN": true, "IRQ": true, "NOR": true,
	// Group J
	"ARG": true, "ALG": true, "AUT": true, "JOR": true,
	// Group K
	"POR": true, "COD": true, "UZB": true, "COL": true,
	// Group L
	"ENG": true, "CRO": true, "GHA": true, "PAN": true,
}

func IsValidGroupCode(code string) bool {
	return validGroupCodes[code]
}

func IsValidStageCode(code string) bool {
	return validStageCodes[code]
}

func IsValidStatus(status string) bool {
	return validStatuses[status]
}

func IsValidFifaCode(code string) bool {
	return validFifaCodes[code]
}

func IsValidDateRange(from, to *time.Time) bool {
	if from == nil || to == nil {
		return true // Only validate if both are provided
	}

	return from.Before(*to) || from.Equal(*to)
}
