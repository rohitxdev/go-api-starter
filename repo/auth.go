package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

const (
	RoleUser  = "USER"
	RoleAdmin = "ADMIN"
)

const (
	GenderMale   = "MALE"
	GenderFemale = "FEMALE"
	GenderOther  = "OTHER"
)

const (
	AccountStatusPending   = "PENDING"
	AccountStatusActive    = "ACTIVE"
	AccountStatusSuspended = "SUSPENDED"
	AccountStatusDeleted   = "DELETED"
)

type User struct {
	Email         string  `json:"email"`
	PasswordHash  string  `json:"passwordHash"`
	Role          string  `json:"role"`
	AccountStatus string  `json:"accountStatus"`
	Username      *string `json:"username,omitempty"`
	ImageURL      *string `json:"imageUrl"`
	Gender        *string `json:"gender,omitempty"`
	DateOfBirth   *string `json:"dateOfBirth"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	ID            uint64  `json:"id"`
}

func (repo *Repo) GetUserById(ctx context.Context, userID uint64) (*User, error) {
	var user User
	err := repo.db.QueryRowContext(ctx, `SELECT id, role, email, password_hash, username, image_url, gender, date_of_birth, account_status, created_at, updated_at FROM auth.users WHERE id=$1 LIMIT 1;`, userID).Scan(&user.ID, &user.Role, &user.Email, &user.PasswordHash, &user.Username, &user.ImageURL, &user.Gender, &user.DateOfBirth, &user.AccountStatus, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		if err, ok := err.(*pq.Error); ok {
			switch code := err.Code.Name(); code {
			case "undefined_column":
				return nil, ErrUserNotFound
			default:
				return nil, fmt.Errorf("Failed to get user by id: %w", err)
			}
		}
		return nil, err
	}
	return &user, nil
}

func (repo *Repo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := repo.db.QueryRowContext(ctx, `SELECT id, role, email, password_hash, username, image_url, gender, date_of_birth, account_status, created_at, updated_at FROM auth.users WHERE email=$1 LIMIT 1;`, email).Scan(&user.ID, &user.Role, &user.Email, &user.PasswordHash, &user.Username, &user.ImageURL, &user.Gender, &user.DateOfBirth, &user.AccountStatus, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		if err, ok := err.(*pq.Error); ok {
			switch code := err.Code.Name(); code {
			case "undefined_column":
				return nil, ErrUserNotFound
			default:
				return nil, fmt.Errorf("Failed to get user by email: %w", err)
			}
		}
		return nil, err
	}
	return &user, nil
}

func (repo *Repo) CreateUser(ctx context.Context, email string, passwordHash string) (uint64, error) {
	var userID uint64
	err := repo.db.QueryRowContext(ctx, `INSERT INTO auth.users (role, email, password_hash, account_status) VALUES($1, $2, $3, $4) RETURNING id;`, RoleUser, email, passwordHash, AccountStatusActive).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *Repo) SetAccountStatusActive(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE auth.users SET account_status=$1 WHERE id=$2;`, AccountStatusActive, id)
	return err
}