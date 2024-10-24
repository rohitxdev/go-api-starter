// Package id provides utility functions for generating unique ids.
package id

import (
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

type prefix uint8

const (
	Request = iota
	User
	Session
)

var prefixes = map[prefix]string{
	Request: "req",
	User:    "usr",
	Session: "ses",
}

func New(prefix prefix) string {
	id := ulid.Make()
	return prefixes[prefix] + "_" + id.String()
}

func Time(id string) (time.Time, error) {
	id = strings.Split(id, "_")[1]
	uid, err := ulid.Parse(id)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse ulid: %w", err)
	}
	return time.UnixMilli(int64(uid.Time())), nil
}
