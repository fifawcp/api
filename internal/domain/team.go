package domain

type Team struct {
	FifaCode  *string `json:"fifa_code"`
	Name      *string `json:"name"`
	FlagURL   *string `json:"flag_url"`
	GroupCode *string `json:"group_code"`
}
