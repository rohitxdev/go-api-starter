// Package id provides utility functions for generating unique ids.
package id

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("create %d id", prefix))
	}
	return prefixes[prefix] + "_" + id.String()
}

func Time(id string) (time.Time, error) {
	id = strings.Split(id, "_")[1]
	uid, err := uuid.Parse(id)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(uid.Time().UnixTime()), nil

}
