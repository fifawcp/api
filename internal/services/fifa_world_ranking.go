package services

// fifaWorldRanking lists FIFA member associations in the order of the most
// recent published edition of the FIFA/Coca-Cola Men's World Ranking, from
// highest to lowest rank

// Used to implement FIFA Article 13 rule g (final tiebreaker after rules a-f):
// when teams cannot be separated by points, head-to-head, overall goal
// difference, overall goals scored, or fair play, they are ranked according
// to their position here
var fifaWorldRanking = []string{
	"FRA", // 1
	"ESP", // 2
	"ARG", // 3
	"ENG", // 4
	"POR", // 5
	"BRA", // 6
	"NED", // 7
	"MAR", // 8
	"BEL", // 9
	"GER", // 10
	"CRO", // 11
	"ITA", // 12
	"COL", // 13
	"SEN", // 14
	"MEX", // 15
	"USA", // 16
	"URU", // 17
	"JPN", // 18
	"SUI", // 19
	"DEN", // 20
	"IRN", // 21
	"TUR", // 22
	"ECU", // 23
	"AUT", // 24
	"KOR", // 25
	"NGA", // 26
	"AUS", // 27
	"ALG", // 28
	"EGY", // 29
	"CAN", // 30
	"NOR", // 31
	"UKR", // 32
	"PAN", // 33
	"CIV", // 34
	"POL", // 35
	"RUS", // 36
	"WAL", // 37
	"SWE", // 38
	"SRB", // 39
	"PAR", // 40
	"CZE", // 41
	"HUN", // 42
	"SCO", // 43
	"TUN", // 44
	"CMR", // 45
	"COD", // 46
	"GRE", // 47
	"SVK", // 48
	"VEN", // 49
	"UZB", // 50
	"CRC", // 51
	"MLI", // 52
	"PER", // 53
	"CHI", // 54
	"QAT", // 55
	"ROU", // 56
	"IRQ", // 57
	"SVN", // 58
	"IRL", // 59
	"RSA", // 60
	"KSA", // 61
	"BFA", // 62
	"JOR", // 63
	"ALB", // 64
	"BIH", // 65
	"HON", // 66
	"MKD", // 67
	"UAE", // 68
	"CPV", // 69
	"NIR", // 70
	"JAM", // 71
	"GEO", // 72
	"FIN", // 73
	"GHA", // 74
	"ISL", // 75
	"BOL", // 76
	"ISR", // 77
	"KOS", // 78
	"OMA", // 79
	"GUI", // 80
	"MNE", // 81
	"CUW", // 82
	"HAI", // 83
	"SYR", // 84
	"NZL", // 85
	"BUL", // 86
	"GAB", // 87
	"UGA", // 88
	"ANG", // 89
	"BEN", // 90
	"BHR", // 91
	"ZAM", // 92
	"THA", // 93
	"CHN", // 94
	"PLE", // 95
	"GUA", // 96
	"BLR", // 97
	"LUX", // 98
	"VIE", // 99
	"SLV", // 100
}

// fifaWorldRankingPosition is a code-to-index lookup over fifaWorldRanking
// Lower index means higher FIFA rank
var fifaWorldRankingPosition = func() map[string]int {
	positions := make(map[string]int, len(fifaWorldRanking))

	for index, fifaCode := range fifaWorldRanking {
		positions[fifaCode] = index
	}

	return positions
}()
