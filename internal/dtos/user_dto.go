package dtos

type CreateUserDto struct {
	Email     string `json:"email" validate:"required,email,max=255" example:"john.doe@example.com"`
	Username  string `json:"username" validate:"required,max=20" example:"johndoe"`
	FirstName string `json:"first_name" validate:"required,max=255" example:"John"`
	LastName  string `json:"last_name" validate:"required,max=255" example:"Doe"`
}

type UpdateUserDto struct {
	FirstName *string `json:"first_name" validate:"omitempty,max=255" example:"John"`
	LastName  *string `json:"last_name" validate:"omitempty,max=255" example:"Doe"`
	Username  *string `json:"username" validate:"omitempty,max=20" example:"johndoe"`
}
