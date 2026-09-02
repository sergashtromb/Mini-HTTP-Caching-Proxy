// this file is needed to define interface

package domain

type CacheStore interface {
	Add(hash string, data []byte)
	Get(hash string) ([]byte, error)
	Delet(hash string)
}