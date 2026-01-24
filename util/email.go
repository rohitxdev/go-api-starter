package util

import "strings"

func MaskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}

	local := email[:at]
	domain := email[at+1:]
	localLen := len(local)

	switch localLen {
	case 1:
		return "*@" + domain
	case 2:
		return local[:1] + "*@" + domain
	default:
		return local[:1] + strings.Repeat("*", localLen-2) + local[localLen-1:] + "@" + domain
	}
}
