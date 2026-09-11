package tools

import (
	"context"
	"hash/fnv"
	"log/slog"
	"runtime/debug"

	"github.com/cespare/xxhash/v2"
)

func ShardIDFromString(s string, qt_shards int) int {

	h := fnv.New32a()
	h.Write([]byte(s))
	
	return int(h.Sum32() % uint32(qt_shards))
}

func ShardIDFromStringxxxHash(s string, qt_shards int) int {

	hash_sum := xxhash.Sum64([]byte(s))

	return int(hash_sum % uint64(qt_shards))

}

func ShardIDFromStringAIMethod(key string, numShards int) int {
	if numShards <= 0 {
		return 0
	}

	var hashVal uint32 = 0

	// Проходим по каждому байту/символу строки (для ASCII/UTF-8)
	for i := 0; i < len(key); i++ {
		// Умножаем на 31 и добавляем код символа.
		// В Go тип uint32 автоматически обрабатывает переполнение так же, как маска & 0xFFFFFFFF
		hashVal = hashVal*31 + uint32(key[i])
	}

	return int(hashVal % uint32(numShards))
}

func SafeGo(rec func(), fn func(ctx context.Context) error, ctx context.Context) {

	go func() {
		defer rec()
		if err := fn(ctx); err != nil {
			slog.Error("Failed SafeGo", "err", err)
		}
		
	}()
	
}

func UnivSafeGo(fn func(ctx context.Context) error, ctx context.Context) {

	SafeGo(func () {
		if r := recover(); r != nil {
			slog.Error("Panic in gorutine", "r", r, "stak", debug.Stack())
		}	
	}, fn, ctx)

}