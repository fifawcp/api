package dtos

import "github.com/fifawcp/api/internal/domain"

type CreateBoardDto struct {
	Name string `json:"name" validate:"required,max=120" example:"League of Dummies"`
}

type JoinBoardDto struct {
	JoinCode string `json:"join_code" validate:"required,min=8,max=8" example:"ABCD1234"`
}

type UpdateBoardDto struct {
	Name string `json:"name" validate:"required,max=120" example:"League of Dummies"`
}

type UpdateBoardMemberRoleDto struct {
	Role domain.BoardMemberRole `json:"role" validate:"required,oneof=member admin" example:"admin"`
}
