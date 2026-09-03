// this file is needed to define interface

package domain

import (
	"context"
	"reflect"
	"time"
)

type CacheStore interface {
	Set(key string, data interface{}, exp time.Duration)
	Get(key string) interface{}
	GetType(key string) reflect.Type
	Del(key string)
	DelExpiration(ctx context.Context)
}