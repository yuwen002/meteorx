package idgen

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

func New() string {
	timestamp := ulid.Timestamp(time.Now())
	entropy := rand.Reader
	id := ulid.MustNew(timestamp, entropy)
	return id.String()
}

func NewUUID() string {
	timestamp := ulid.Timestamp(time.Now())
	entropy := rand.Reader
	id := ulid.MustNew(timestamp, entropy)
	return id.String()
}

func MustParse(s string) ulid.ULID {
	return ulid.MustParse(s)
}

func Parse(s string) (ulid.ULID, error) {
	return ulid.Parse(s)
}