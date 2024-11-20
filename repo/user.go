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

type User struct {
	Email         string  `json:"email"`
	Role          string  `json:"role"`
	AccountStatus string  `json:"accountStatus"`
	FullName      *string `json:"fullName,omitempty"`
	DateOfBirth   *string `json:"dateOfBirth"`
	Gender        *string `json:"gender,omitempty"`
	PhoneNumber   *string `json:"phoneNumber,omitempty"`
	ImageUrl      *string `json:"imageUrl"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	Id            uint64  `json:"id"`
	IsVerified    bool    `json:"isVerified"`
}

func (repo *Repo) GetUserById(ctx context.Context, userId uint64) (*User, error) {
	var user User
	err := repo.db.QueryRowContext(ctx, `SELECT id, role, email, full_name, date_of_birth, gender, phone_number, account_status, image_url, is_verified, created_at, updated_at FROM users WHERE id=$1 LIMIT 1;`, userId).Scan(&user.Id, &user.Role, &user.Email, &user.FullName, &user.DateOfBirth, &user.Gender, &user.PhoneNumber, &user.AccountStatus, &user.ImageUrl, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)

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
	err := repo.db.QueryRowContext(ctx, `SELECT id, role, email, full_name, date_of_birth, gender, phone_number, account_status, image_url, is_verified, created_at, updated_at FROM users WHERE email=$1 LIMIT 1;`, email).Scan(&user.Id, &user.Role, &user.Email, &user.FullName, &user.DateOfBirth, &user.Gender, &user.PhoneNumber, &user.AccountStatus, &user.ImageUrl, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)

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

func (repo *Repo) CreateUser(ctx context.Context, email string) (uint64, error) {
	var userId uint64
	err := repo.db.QueryRowContext(ctx, `INSERT INTO users(email) VALUES($1) RETURNING id;`, email).Scan(&userId)
	if err != nil {
		return 0, err
	}
	return userId, nil
}

func (repo *Repo) Update(ctx context.Context, id string, updates map[string]any) error {
	query := "UPDATE users SET "
	var params []interface{}

	count := 1
	for key, value := range updates {
		query += fmt.Sprintf("%s=$%v, ", key, count)
		params = append(params, value)
		count++
	}

	// Remove the trailing comma and space
	query = query[:len(query)-2]

	query += fmt.Sprintf(" WHERE id=$%v;", count)
	params = append(params, id)
	_, err := repo.db.Exec(query, params...)
	return err
}

func (r *Repo) SetIsVerified(ctx context.Context, id uint64, isVerified bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET is_verified=$1 WHERE id=$2;`, isVerified, id)
	return err
}
