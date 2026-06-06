package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func boardsRoutes(c *Container) chi.Router {
	r := chi.NewRouter()

	r.Get("/preview", c.BoardHandler.GetBoardPreview)

	r.Group(func(r chi.Router) {
		r.Use(middlewares.Auth(c.Authenticator, c.UserService, c.Logger))

		r.Get("/", c.BoardHandler.GetUserBoards)
		r.Post("/", c.BoardHandler.CreateBoard)
		r.Post("/join", c.BoardHandler.JoinBoard)

		r.Route("/{boardId}", func(r chi.Router) {
			r.Use(middlewares.RequireBoardMembership(c.BoardMemberService))

			r.Get("/", c.BoardHandler.GetBoardByID)
			r.Patch("/", c.BoardHandler.UpdateBoard)
			r.Delete("/", c.BoardHandler.DeleteBoard)
			r.Delete("/leave", c.BoardHandler.LeaveBoard)
			r.Post("/regenerate-join-code", c.BoardHandler.RegenerateJoinCode)

			r.Route("/members", func(r chi.Router) {
				r.Get("/", c.BoardHandler.GetBoardMembers)

				r.Route("/{userId}", func(r chi.Router) {
					r.Use(middlewares.ParseUserID)
					r.Patch("/role", c.BoardHandler.UpdateBoardMemberRole)
					r.Delete("/", c.BoardHandler.RemoveBoardMember)
					r.Post("/transfer-ownership", c.BoardHandler.TransferOwnership)
				})
			})

			r.Get("/summary", c.CompetitionHandler.GetBoardSummary)

			r.Route("/competitions", func(r chi.Router) {
				r.Get("/", c.CompetitionHandler.GetBoardCompetitions)
				r.Post("/", c.CompetitionHandler.CreateCompetition)

				r.Route("/{competitionId}", func(r chi.Router) {
					r.Use(middlewares.ParseCompetitionID)
					r.Delete("/", c.CompetitionHandler.DeleteCompetition)
					r.Get("/leaderboard", c.CompetitionHandler.GetLeaderboard)
				})
			})
		})
	})

	return r
}
