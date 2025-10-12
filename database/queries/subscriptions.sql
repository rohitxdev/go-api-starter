-- name: GetSubscriptionByAccountID :one
SELECT * FROM subscriptions WHERE account_id = @account_id;