package dtos

type SaveGroupPicksDto struct {
	GroupPicks []GroupPickDto `json:"group_picks" validate:"required,len=12,dive"`
}

type GroupPickDto struct {
	GroupCode     string   `json:"group_code" validate:"required,len=1,oneof=A B C D E F G H I J K L" example:"A"`
	TeamFifaCodes []string `json:"team_fifa_codes" validate:"required,len=4,unique,dive,required,fifa_code" example:"MEX,RSA,KOR,CZE"` // index 0 = pos 1, index 3 = pos 4
}

type SetGroupLockDto struct {
	GroupCode     string   `json:"group_code" validate:"required,len=1,oneof=A B C D E F G H I J K L" example:"A"`
	Locked        bool     `json:"locked" example:"true"`
	TeamFifaCodes []string `json:"team_fifa_codes" validate:"required,len=4,unique,dive,required,fifa_code" example:"MEX,RSA,KOR,CZE"`
}

type SaveBestThirdsDto struct {
	TeamFifaCodes []string `json:"team_fifa_codes" validate:"required,len=8,unique,dive,required,fifa_code" example:"MEX,RSA,KOR,CZE,BRA,MAR,HAI,SCO"`
}

type SaveBracketPicksDto struct {
	BracketPicks []BracketPickDto `json:"bracket_picks" validate:"required,len=32,unique,dive"`
}

type BracketPickDto struct {
	MatchID      int64  `json:"match_id" validate:"required,min=73,max=104" example:"73"`
	TeamFifaCode string `json:"team_fifa_code" validate:"required,fifa_code" example:"MEX"`
}
