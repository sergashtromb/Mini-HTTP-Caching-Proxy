// this file is needed to define interface

package domain


type ShardLimiter interface {
	Allow(ip string) bool
	Wait(ip string) error
}