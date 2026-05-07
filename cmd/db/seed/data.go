package main

const (
	usersAmount          = 300
	boardsAmount         = 20
	boardMembersMin      = 5
	boardMembersMax      = 50
	pickemUserPercentage = 0.60
	boardNameMaxLength   = 20
	usernameMaxLength    = 20
)

var teamsByGroup = map[string][]string{
	"A": {"MEX", "RSA", "KOR", "CZE"},
	"B": {"CAN", "BIH", "QAT", "SUI"},
	"C": {"BRA", "MAR", "HAI", "SCO"},
	"D": {"USA", "PAR", "AUS", "TUR"},
	"E": {"GER", "CUW", "CIV", "ECU"},
	"F": {"NED", "JPN", "SWE", "TUN"},
	"G": {"BEL", "EGY", "IRN", "NZL"},
	"H": {"ESP", "CPV", "KSA", "URU"},
	"I": {"FRA", "SEN", "IRQ", "NOR"},
	"J": {"ARG", "ALG", "AUT", "JOR"},
	"K": {"POR", "COD", "UZB", "COL"},
	"L": {"ENG", "CRO", "GHA", "PAN"},
}

var groupStageMatchIDs = func() []int64 {
	ids := make([]int64, 72)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	return ids
}()
