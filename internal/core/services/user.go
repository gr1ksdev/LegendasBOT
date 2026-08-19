package services

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type UserService struct {
	userRepo *repositories.UserRepository
}

func NewUserService(userRepo *repositories.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) UpsertUser(ctx context.Context, user *models.User) error {
	if err := s.userRepo.UpsertUser(ctx, user); err != nil {
		return errors.Internal(err)
	}
	return nil
}

func (s *UserService) GetAllUsersPaginated(ctx context.Context, limit, offset int) ([]models.User, int64, error) {
	users, total, err := s.userRepo.GetAllUsersPaginated(ctx, limit, offset)
	if err != nil {
		return nil, 0, errors.Internal(err)
	}
	return users, total, nil
}

func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	users, err := s.userRepo.GetAllUsers(ctx)
	if err != nil {
		return nil, errors.Internal(err)
	}
	return users, nil
}

func (s *UserService) GetAllUsersWithChannels(ctx context.Context) ([]models.User, error) {
	users, err := s.userRepo.GetAllUsersWithChannels(ctx)
	if err != nil {
		return nil, errors.Internal(err)
	}
	return users, nil
}

func (s *UserService) UpdateUserAdmin(ctx context.Context, userID int64) (bool, error) {
	isAdmin, err := s.userRepo.UpdateUserAdmin(ctx, userID)
	if err != nil {
		return false, errors.Internal(err)
	}
	return isAdmin, nil
}

func (s *UserService) UpdateUserBlacklist(ctx context.Context, userID int64) (bool, error) {
	isBlacklisted, err := s.userRepo.UpdateUserBlacklist(ctx, userID)
	if err != nil {
		return false, errors.Internal(err)
	}
	return isBlacklisted, nil
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, errors.ErrNotFound
	}
	return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.userRepo.GetUserById(ctx, userID)
	if err != nil {
		return nil, errors.ErrNotFound
	}
	return user, nil
}
