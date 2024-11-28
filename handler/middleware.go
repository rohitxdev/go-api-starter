package handler

type role string

const (
	RoleUser  role = "user"
	RoleAdmin role = "admin"
)

var roles = map[role]uint8{
	RoleUser:  1,
	RoleAdmin: 2,
}
