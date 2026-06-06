package services

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type CompetitionScoringServiceInterface interface {
	RecomputeForMatches(ctx context.Context, result *domain.ScoreMatchesResult) error
}

type CompetitionScoringService struct {
	competitionScoreRepository domain.CompetitionScoreRepository
	cfg                        *config.Config
	logger                     logging.Logger
}

func NewCompetitionScoringService(
	competitionScoreRepository domain.CompetitionScoreRepository,
	cfg *config.Config,
	logger logging.Logger,
) CompetitionScoringServiceInterface {
	return &CompetitionScoringService{
		competitionScoreRepository: competitionScoreRepository,
		cfg:                        cfg,
		logger:                     logger,
	}
}

// RecomputeForMatches refreshes the match/pick score caches for competitions
// affected by a scoring run. Pick'em and awards leaderboards are computed on
// read, so they need no recompute.
func (s *CompetitionScoringService) RecomputeForMatches(ctx context.Context, result *domain.ScoreMatchesResult) error {
	if len(result.AffectedUserIDs) == 0 || len(result.ScoredMatchIDs) == 0 {
		return nil
	}

	competitionIDs, err := s.competitionScoreRepository.FindMatchCompetitionsByMatches(ctx, result.ScoredMatchIDs)
	if err != nil {
		s.logger.Error("failed to find match competitions",
			logging.Error, err.Error(),
			"match_ids", result.ScoredMatchIDs,
		)
		return err
	}

	for _, competitionID := range competitionIDs {
		if err := s.competitionScoreRepository.BatchUpsertMatchScores(
			ctx, competitionID, result.AffectedUserIDs, s.cfg.Scoring.MatchScoreExact,
		); err != nil {
			s.logger.Error("failed to upsert match scores",
				logging.Error, err.Error(),
				"competition_id", competitionID,
			)
			return err
		}
	}

	pickIDs, err := s.competitionScoreRepository.FindPickCompetitionsByMatches(ctx, result.ScoredMatchIDs)
	if err != nil {
		s.logger.Error("failed to find pick competitions",
			logging.Error, err.Error(),
			"match_ids", result.ScoredMatchIDs,
		)
		return err
	}

	for _, competitionID := range pickIDs {
		if err := s.competitionScoreRepository.BatchUpsertPickScores(
			ctx, competitionID, result.AffectedUserIDs, s.cfg.Scoring.MatchScoreExact,
		); err != nil {
			s.logger.Error("failed to upsert pick scores",
				logging.Error, err.Error(),
				"competition_id", competitionID,
			)
			return err
		}
	}

	return nil
}
