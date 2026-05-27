package main

const (
	usersAmount        = 300
	boardsAmount       = 40
	boardMembersMin    = 5
	boardMembersMax    = 50
	boardNameMaxLength = 20
	usernameMaxLength  = 20
)

// EngagementMix describes how seeded users split across the four disjoint
// engagement buckets. All four fields must sum to 1.0.
//
//	PickemAndMatch — most engaged: predicted the bracket AND scored matches
//	PickemOnly     — predicted the bracket but skipped match scores
//	MatchOnly      — casual match-by-match guessers, no bracket prediction
//	Idle           — created accounts but never interacted with picks
type EngagementMix struct {
	PickemAndMatch float64
	PickemOnly     float64
	MatchOnly      float64
	Idle           float64
}

var userEngagement = EngagementMix{
	PickemAndMatch: 0.40,
	PickemOnly:     0.20,
	MatchOnly:      0.30,
	Idle:           0.10,
}

var boardNames = []string{
	"The Offside Crew",
	"Golden Boot Club",
	"Penalty Kings",
	"The Net Busters",
	"Upper 90 Gang",
	"The Ultras",
	"Hat Trick Heroes",
	"Dead Ball Society",
	"The Nutmeg Club",
	"Free Kick Union",
	"Late Tackle FC",
	"High Press Gang",
	"Gegenpressing Co",
	"Parking the Bus",
	"Sweeper Keepers",
	"Box to Box Crew",
	"False 9 Society",
	"The Target Men",
	"Inverted Wingers",
	"Half Space Heroes",
	"Zonal Markers",
	"The Libero Club",
	"The Shadow Nine",
	"The Pivot Bros",
	"Double Pivot FC",
	"Channel Runners",
	"Set Piece Squad",
	"Dead Ball Kings",
	"Wing Play United",
	"The Crossers",
	"Near Post Gang",
	"Second Ball Boys",
	"Midfield Masters",
	"Engine Room FC",
	"Box Crashers",
	"The Late Runners",
	"Cutback Crew",
	"One Two Club",
	"Press Trap FC",
	"Long Rangers",
	"Strike Force",
	"Clinical FC",
	"The Goal Hounds",
	"Header Kings",
	"The Aerial Crew",
	"Hold Up Boys",
	"Link Up FC",
	"The Back Four",
	"Regista FC",
	"The Wide Men",
	"Los Ultras",
	"La Banda Goleadora",
	"El Equipo Loco",
	"Los Crack FC",
	"La Hinchada",
	"El Toque Final",
	"Los Rematadores",
	"La Punta del Gol",
	"Los Gambeteadores",
	"El Pelotazo",
	"Los Cañoneros",
	"La Pelota Loca",
	"Los Penalteros",
	"La Chilena FC",
	"Los Caños FC",
	"El Contraataque",
	"Los Tapones",
	"La Pared United",
	"Los Regates",
	"El Pivote Club",
	"Los Mediapuntas",
	"Los Delanteros",
	"El Cabezazo FC",
	"Los Liberos",
	"La Volea Club",
	"Los Goleadores",
	"El Sombrero FC",
	"Los Extremos",
	"La Gambeta Club",
	"Los Zagueros",
	"El Doble Pivote",
	"Los Arqueros",
	"La Vaselina FC",
	"El Enganche",
	"Los Volantes",
	"La Patada Seca",
	"Los Finalizadores",
	"El Área Chica",
	"La Pelusa Club",
	"Los Kamikazes",
	"El Primer Palo",
	"La Puerta Grande",
	"Los Fantasistas",
	"El Medio Centro",
	"Los Cracks",
	"La Presión Alta",
	"Los Números 10",
	"Los Taloneros",
	"La Pared FC",
	"El Offside Club",
}

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
