package dtos

import "github.com/fifawcp/api/internal/domain"

type CreateBoardDto struct {
	Name string `json:"name" validate:"required,max=20" example:"League of Dummies"`
}

type JoinBoardDto struct {
	JoinCode string `json:"join_code" validate:"required,min=8,max=8" example:"ABCD1234"`
}

type JoinBoardResponseDto struct {
	BoardID string `json:"board_id" example:"123e4567-e89b-12d3-a456-426614174000"`
}

type UpdateBoardDto struct {
	Name string `json:"name" validate:"required,max=20" example:"League of Dummies"`
}

type UpdateBoardMemberRoleDto struct {
	Role domain.BoardMemberRole `json:"role" validate:"required,oneof=member admin" example:"admin"`
}
