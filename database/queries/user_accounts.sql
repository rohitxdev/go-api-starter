-- name: GetUserAccountsByUserID :many
SELECT a.* FROM accounts AS a JOIN user_accounts as ua ON ua.account_id = a.id WHERE ua.user_id = @user_id;