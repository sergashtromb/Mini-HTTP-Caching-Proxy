package test

import (
	"mini_http_caching_proxy/tools"
	"testing"
)

func BenchmarkHashSumxHash(b *testing.B) {
	
	str := "one/two/{param1}192.255.45.1"
	shards := 8

	for i := 0; i < b.N; i++ {
		_ = tools.ShardIDFromStringxxxHash(str, shards)
	}

}

func BenchmarkHashSumFNVAlg(b *testing.B) {
	
	str := "one/two/{param1}192.255.45.1"
	shards := 8

	for i := 0; i < b.N; i++ {
		_ = tools.ShardIDFromString(str, shards)
	}

}

func BenchmarkHashSumAIMethod(b *testing.B) {
	
	str := "one/two/{param1}192.255.45.1"
	shards := 8

	for i := 0; i < b.N; i++ {
		_ = tools.ShardIDFromString(str, shards)
	}

}