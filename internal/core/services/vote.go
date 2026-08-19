package services

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type VoteService struct {
	voteRepo *repositories.VoteRepository
}

func NewVoteService(voteRepo *repositories.VoteRepository) *VoteService {
	return &VoteService{voteRepo: voteRepo}
}

func (s *VoteService) ToggleVote(ctx context.Context, chatID int64, messageID int, inlineMessageID string, userID int64, emoji string) (bool, int64, error) {
	added, count, err := s.voteRepo.ToggleVote(ctx, chatID, messageID, inlineMessageID, userID, emoji)
	if err != nil {
		return false, 0, errors.Internal(err)
	}
	return added, count, nil
}

func (s *VoteService) GetVoteCounts(ctx context.Context, chatID int64, messageID int, inlineMessageID string) (map[string]int64, error) {
	counts, err := s.voteRepo.GetVoteCounts(ctx, chatID, messageID, inlineMessageID)
	if err != nil {
		return nil, errors.Internal(err)
	}
	return counts, nil
}
