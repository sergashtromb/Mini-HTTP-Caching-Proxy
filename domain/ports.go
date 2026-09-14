// this file is needed to define interface

package domain

import (
	"time"
)

type CacheStore interface {
	Set(key string, data []byte) error
	SetWithExp(key string, data []byte, exp time.Duration) error
	Get(key string) ([]byte, error)
}