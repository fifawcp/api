package services

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

type CompetitionScoringServiceInterface interface {
	RecomputeForMatches(ctx context.Context, result *domain.ScoreMatchesResult) error
	RecomputeForBestThirds(ctx context.Context, affectedUserIDs []string) error
	RecomputeForAwards(ctx context.Context, affectedUserIDs []string) error
}

type CompetitionScoringService struct {
	competitionRepository      domain.CompetitionRepository
	competitionScoreRepository domain.CompetitionScoreRepository
	cfg                        *config.Config
	logger                     logging.Logger
}

func NewCompetitionScoringService(
	competitionRepository domain.CompetitionRepository,
	competitionScoreRepository domain.CompetitionScoreRepository,
	cfg *config.Config,
	logger logging.Logger,
) CompetitionScoringServiceInterface {
	return &CompetitionScoringService{
		competitionRepository:      competitionRepository,
		competitionScoreRepository: competitionScoreRepository,
		cfg:                        cfg,
		logger:                     logger,
	}
}

// RecomputeForMatches recomputes the per-type score caches for all competitions
// affected by a scoring run. Called after ScoreMatches completes
func (s *CompetitionScoringService) RecomputeForMatches(ctx context.Context, result *domain.ScoreMatchesResult) error {
	if len(result.AffectedUserIDs) == 0 {
		return nil
	}

	// Match competitions
	if len(result.ScoredMatchIDs) > 0 {
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
	}

	// Pool competitions (single-match pools share the match-score cache)
	if len(result.ScoredMatchIDs) > 0 {
		poolIDs, err := s.competitionScoreRepository.FindPoolCompetitionsByMatches(ctx, result.ScoredMatchIDs)
		if err != nil {
			s.logger.Error("failed to find pool competitions",
				logging.Error, err.Error(),
				"match_ids", result.ScoredMatchIDs,
			)
			return err
		}

		for _, competitionID := range poolIDs {
			if err := s.competitionScoreRepository.BatchUpsertPoolScores(
				ctx, competitionID, result.AffectedUserIDs, s.cfg.Scoring.MatchScoreExact,
			); err != nil {
				s.logger.Error("failed to upsert pool scores",
					logging.Error, err.Error(),
					"competition_id", competitionID,
				)
				return err
			}
		}
	}

	// Pickem competitions
	if result.PickemAffected {
		if err := s.recomputeAllPickems(ctx, result.AffectedUserIDs); err != nil {
			return err
		}
	}

	return nil
}

func (s *CompetitionScoringService) RecomputeForBestThirds(ctx context.Context, affectedUserIDs []string) error {
	if len(affectedUserIDs) == 0 {
		return nil
	}

	return s.recomputeAllPickems(ctx, affectedUserIDs)
}

func (s *CompetitionScoringService) RecomputeForAwards(ctx context.Context, affectedUserIDs []string) error {
	if len(affectedUserIDs) == 0 {
		return nil
	}

	return s.recomputeAllPickems(ctx, affectedUserIDs)
}

func (s *CompetitionScoringService) recomputeAllPickems(ctx context.Context, userIDs []string) error {
	competitionIDs, err := s.competitionRepository.GetAllPickemIDs(ctx)
	if err != nil {
		s.logger.Error("failed to get pickem competition IDs",
			logging.Error, err.Error(),
		)
		return err
	}

	if len(competitionIDs) == 0 {
		return nil
	}

	if err := s.competitionScoreRepository.BatchUpsertPickemScores(ctx, competitionIDs, userIDs); err != nil {
		s.logger.Error("failed to upsert pickem scores",
			logging.Error, err.Error(),
			"competition_count", len(competitionIDs),
			"user_count", len(userIDs),
		)
		return err
	}

	return nil
}
