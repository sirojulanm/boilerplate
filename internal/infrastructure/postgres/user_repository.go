package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"gorm.io/gorm"
)

// UserRepository implements both domain.UserCreator and domain.UserQuery using
// GORM. Every database operation goes through GORM's query builder.
type UserRepository struct {
	db *gorm.DB
}

var (
	_ domain.UserCreator = (*UserRepository)(nil)
	_ domain.UserQuery   = (*UserRepository)(nil)
)

// NewUserRepository wires a GORM handle into the repository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new account. Duplicate emails become domain.ErrEmailTaken.
func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	return mapGormError(r.db.WithContext(ctx).Create(&user).Error)
}

// FindByEmail looks up an account by its normalized email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("email = ?", domain.NormalizeEmail(email)).
		First(&user).Error
	if err != nil {
		return domain.User{}, mapGormError(err)
	}
	return user, nil
}

// FindByID looks up an account by primary key.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return domain.User{}, mapGormError(err)
	}
	return user, nil
}

func mapGormError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrUserNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrEmailTaken
	}

	// Fallback for drivers or GORM versions that do not translate the error.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrEmailTaken
	}
	return err
}
