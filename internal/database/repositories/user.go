package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) UpsertUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"first_name", "username", "updated_at"}),
	}).Create(user).Error
}

func (r *UserRepository) GetAllUsersPaginated(ctx context.Context, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	db := r.db.WithContext(ctx).Model(&models.User{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&users).Error
	return users, err
}

func (r *UserRepository) GetAllUsersWithChannels(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Preload("Channels").Order("created_at DESC").Find(&users).Error
	return users, err
}

func (r *UserRepository) GetUserById(ctx context.Context, userID int64) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	return &user, err
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *UserRepository) UpdateUserAdmin(ctx context.Context, userID int64) (bool, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error; err != nil {
		return false, err
	}

	newValue := !user.IsAdmin
	err := r.db.WithContext(ctx).Model(&user).Update("is_admin", newValue).Error
	return newValue, err
}

// UpdateFeatures atualiza apenas o campo features de um usuario.
func (r *UserRepository) UpdateFeatures(ctx context.Context, userID int64, features string) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("user_id = ?", userID).
		Update("features", features).Error
}

func (r *UserRepository) UpdateUserBlacklist(ctx context.Context, userID int64) (bool, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error; err != nil {
		return false, err
	}

	newValue := !user.IsBlacklisted
	err := r.db.WithContext(ctx).Model(&user).Update("is_blacklisted", newValue).Error
	return newValue, err
}
