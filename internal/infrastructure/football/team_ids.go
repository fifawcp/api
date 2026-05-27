package football

var APITeamIDToFIFACode = map[int64]string{
	// Group A
	16:   "MEX",
	17:   "KOR",
	770:  "CZE",
	1531: "RSA",
	// Group B
	15:   "SUI",
	1113: "BIH",
	1569: "QAT",
	5529: "CAN",
	// Group C
	6:    "BRA",
	31:   "MAR",
	1108: "SCO",
	2386: "HAI",
	// Group D
	20:   "AUS",
	777:  "TUR",
	2380: "PAR",
	2384: "USA",
	// Group E
	25:   "GER",
	1501: "CIV",
	2382: "ECU",
	5530: "CUW",
	// Group F
	12:   "JPN",
	28:   "TUN",
	5:    "SWE",
	1118: "NED",
	// Group G
	1:    "BEL",
	32:   "EGY",
	22:   "IRN",
	4673: "NZL",
	// Group H
	9:    "ESP",
	23:   "KSA",
	7:    "URU",
	1533: "CPV",
	// Group I
	2:    "FRA",
	13:   "SEN",
	1090: "NOR",
	1567: "IRQ",
	// Group J
	26:   "ARG",
	775:  "AUT",
	1532: "ALG",
	1548: "JOR",
	// Group K
	8:    "COL",
	27:   "POR",
	1508: "COD",
	1568: "UZB",
	// Group L
	10:   "ENG",
	11:   "PAN",
	3:    "CRO",
	1504: "GHA",
}

var FifaCodeToAPITeamID map[string]int64

func init() {
	FifaCodeToAPITeamID = make(map[string]int64, len(APITeamIDToFIFACode))
	for apiID, code := range APITeamIDToFIFACode {
		FifaCodeToAPITeamID[code] = apiID
	}
}
