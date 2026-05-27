package domain

type Pagination struct {
	Page    int  `json:"page" example:"1"`
	Limit   int  `json:"limit" example:"20"`
	Total   int  `json:"total" example:"42"`
	HasMore bool `json:"has_more" example:"true"`
}
