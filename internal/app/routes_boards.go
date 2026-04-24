package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func boardsRoutes(c *Container) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares.Auth(c.Authenticator, c.UserService, c.Logger))

	r.Get("/", c.BoardHandler.GetUserBoards)
	r.Post("/", c.BoardHandler.CreateBoard)
	r.Post("/join", c.BoardHandler.JoinBoard)

	r.Route("/{boardId}", func(r chi.Router) {
		r.Use(middlewares.RequireBoardMembership(c.BoardMemberService))

		r.Get("/", c.BoardHandler.GetBoardByID)
		r.Patch("/", c.BoardHandler.UpdateBoard)
		r.Delete("/", c.BoardHandler.DeleteBoard)
		r.Get("/ranking", c.BoardHandler.GetBoardRanking)
		r.Post("/regenerate-join-code", c.BoardHandler.RegenerateJoinCode)

		r.Route("/members", func(r chi.Router) {
			r.Get("/", c.BoardHandler.GetBoardMembers)
			r.With(middlewares.RequireValidUserID).Patch("/{userId}/role", c.BoardHandler.UpdateBoardMemberRole)
			r.With(middlewares.RequireValidUserID).Delete("/{userId}", c.BoardHandler.RemoveBoardMember)
		})
	})

	return r
}
