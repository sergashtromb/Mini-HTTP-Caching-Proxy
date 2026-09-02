package main

import (
	"fmt"
	"math/rand"
	"mini_http_caching_proxy/tools"
	//"slices"
)

func main() {

	//qt_shard := 20

	final_arr := make([][]int64, 3)

	for i := 0; i < 3; i++ {
		final_arr[i] = make([]int64, 1000)
	}

	paths := []string{"one", "two", "three", "four", "{param1}", "{old1}", "five"}

	shards := []int{2, 8, 6, 10, 20, 12, 16, 4, 14, 18}

	qt1 := 0
	qt2 := 0
	qt3 := 0

	for i := range 200 {

		rn_shard := shards[rand.Intn(len(shards)-1)]
		iterations := 1000 + rand.Intn(10000-1000+1)

		arr := make([]int64, rn_shard)
		arr2 := make([]int64, rn_shard)
		arr3 := make([]int64, rn_shard)

		for range iterations {

			path := paths[rand.Intn(6)] + "/" + paths[rand.Intn(6)] + "/" + paths[rand.Intn(6)] + "/" + paths[rand.Intn(6)] + "/" + paths[rand.Intn(6)] + "/"
			ip := string(rand.Intn(255)) + "." + string(rand.Intn(255)) + "." + string(rand.Intn(255)) + "." + string(rand.Intn(255)) + "."

			fullstring := path + ip

			arr[tools.ShardIDFromString(fullstring, rn_shard)] += 1
			arr2[tools.ShardIDFromStringxxxHash(fullstring, rn_shard)] += 1
			arr3[tools.ShardIDFromStringAIMethod(fullstring, rn_shard)] += 1

		}

		final_arr[0][i] = arr[len(arr) - 1] - arr[0]
		final_arr[1][i] = arr2[len(arr2) - 1] - arr2[0]
		final_arr[2][i] = arr3[len(arr3) - 1] - arr3[0]

		minq := min(final_arr[0][i], final_arr[1][i], final_arr[2][i])

		if minq == final_arr[0][i] {
			qt1++
		} else if minq == final_arr[1][i] {
			qt2++
		} else if minq == final_arr[2][i] {
			qt3++
		}

	}

	fmt.Printf("FNV     - %v\nxxxHash - %v\nAI      - %v\n", qt1, qt2, qt3)

}