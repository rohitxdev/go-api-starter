package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/rohitxdev/go-api-starter/internal/id"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

type User struct {
	Email         string `json:"email"`
	Role          string `json:"role"`
	FullName      string `json:"full_name,omitempty"`
	Username      string `json:"username,omitempty"`
	DateOfBirth   string `json:"date_of_birth"`
	Gender        string `json:"gender,omitempty"`
	PhoneNumber   string `json:"phone_number,omitempty"`
	AccountStatus string `json:"account_status"`
	ImageUrl      string `json:"image_url"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	Id            string `json:"id"`
	IsVerified    bool   `json:"is_verified"`
}

func (repo *Repo) GetUserById(ctx context.Context, userId string) (*User, error) {
	user := new(User)
	err := repo.db.QueryRowContext(ctx, `SELECT id, role, email, COALESCE(username, ''), COALESCE(full_name, ''), COALESCE(date_of_birth, '-infinity'), COALESCE(gender, ''), COALESCE(phone_number, ''), COALESCE(account_status, ''), COALESCE(image_url, ''), is_verified, created_at, updated_at FROM users WHERE id=$1 LIMIT 1;`, userId).Scan(&user.Id, &user.Role, &user.Email, &user.Username, &user.FullName, &user.DateOfBirth, &user.Gender, &user.PhoneNumber, &user.AccountStatus, &user.ImageUrl, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		if err, ok := err.(*pq.Error); ok {
			switch code := err.Code.Name(); code {
			case "undefined_column":
				return nil, ErrUserNotFound
			default:
				return nil, errors.New(code)
			}
		}
		return nil, err
	}
	return user, nil
}

func (repo *Repo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := new(User)
	err := repo.db.QueryRowContext(ctx, `SELECT id, role, email, COALESCE(username, ''), COALESCE(full_name, ''), COALESCE(date_of_birth, '-infinity'), COALESCE(gender, ''), COALESCE(phone_number, ''), COALESCE(account_status, ''), COALESCE(image_url, ''), is_verified, created_at, updated_at FROM users WHERE email=$1 LIMIT 1;`, email).Scan(&user.Id, &user.Role, &user.Email, &user.Username, &user.FullName, &user.DateOfBirth, &user.Gender, &user.PhoneNumber, &user.AccountStatus, &user.ImageUrl, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		if err, ok := err.(*pq.Error); ok {
			switch code := err.Code.Name(); code {
			case "undefined_column":
				return nil, ErrUserNotFound
			default:
				return nil, errors.New(code)
			}
		}
		return nil, err
	}
	return user, nil
}

func (repo *Repo) CreateUser(ctx context.Context, email string) (string, error) {
	userId := id.New(id.User)
	err := repo.db.QueryRowContext(ctx, `INSERT INTO users(id, email) VALUES($1, $2) RETURNING id;`, userId, email).Scan(&userId)
	if err != nil {
		return "", err
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

func (r *Repo) SetIsVerified(ctx context.Context, id string, isVerified bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET is_verified=$1 WHERE id=$2;`, isVerified, id)
	return err
}
