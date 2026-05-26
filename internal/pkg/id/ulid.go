package id

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

func New() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

func IsValid(s string) bool {
	if len(s) != 26 {
		return false
	}
	_, err := ulid.Parse(s)
	return err == nil
}
