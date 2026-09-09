package test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"mini_http_caching_proxy/initial/stores"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)


func TestRecordStructFuncsFromNewRecord(t *testing.T) {
	
	my_key := "my_key"
	data := []byte(my_key)

	rec, size := stores.NewRecord(my_key, data)

	if string(rec.Key) != my_key {
		t.Errorf("Failed create record, keys don't equals rec.Key=%s and my_key=%s", string(rec.Key), my_key)
	}

	if rec.KeyLen != uint32(len(data)) {
		t.Errorf("Failed create record, keys len don't equals rec.KeyLen=%v and len my_key=%v", rec.KeyLen, uint32(len(data)))
	}

	if size != 21 {
		t.Errorf("Failed create record, size returns from NewRecord don't equals 21 size=%v", size)
	}

	if rec.RecSize() != 21 {
		t.Errorf("Failed create record, rec.RecSize() returns val don't equals 21 size=%v", rec.RecSize())
	}

	if my_key != rec.GetKey() {
		t.Errorf("Failed create record, rec.GetKey() returns val don't equals my_key my_key=%v, GetKey()=%v", my_key, rec.GetKey())
	}

	if !slices.Equal(data, *rec.GetData()) {
		t.Errorf("Failed create record, rec.GetData() returns *val don't equals data data=%v, *GetData()=%v", data, *rec.GetData())
	}
}

func TestIndexRecordStructFuncsFromNewIndexRecord(t *testing.T) {

	my_key := "my_key"
	data := []byte(my_key)

	rec, _ := stores.NewRecord(my_key, data)
	file_size := 128

	now := time.Now()
	nowPlus10 := now.Add(10*time.Second)
	nowMinus10 := now.Add(15*time.Second)

	idx_rec, new_fl_size := stores.NewIndexRecordFromRecord(&rec, int64(file_size), nowPlus10.Unix())

	// file_size + rec.RecSize() 128 + 21
	if new_fl_size != 149 {
		t.Errorf("Failed creat index record from NewIndexRecordFromRecord, new_fl_size don't equal 149 new_fl_size=%v", new_fl_size)
	}
	// file_size
	if idx_rec.GetOffset() != 128 {
		t.Errorf("Failed creat index record from NewIndexRecordFromRecord, idx_rec.GetOffset() don't equal 128(file_size) GetOffset()=%v", idx_rec.GetOffset())
	}

	time.Sleep(11*time.Second)

	if !idx_rec.IsDead() {
		t.Errorf("Failed check is dead index record from NewIndexRecordFromRecord, index record don't dead")
	}

	if !idx_rec.IsDeadFromTime(nowMinus10) {
		t.Errorf("Failed check is dead index record from NewIndexRecordFromRecord, index record don't dead from time tm=%v",
			nowMinus10.Sub(time.Unix(idx_rec.Expiration, 0)).Seconds())
	}
}

func TestFileShardBaseFuncs(t *testing.T) {

	tmp := `C:\Temp\proxy`

	fs, err := stores.NewFileShard(&tmp, 100000)
	if err != nil {
		fmt.Errorf("Failed new file shard err=%v", err)
	}

	key1 := "my_data_1"
	data1 := []byte("my data 1")
	key2 := "key_2"
	data2 := []byte{10, 250, 100}
	key3 := "key_3_del"
	data3 := make([]byte, 10)

	err = fs.Set(key1, time.Now().Add(20*time.Minute).Unix(), data1)
	if err != nil {
		fmt.Errorf("Failed set data 1 err=%v", err)
	}
	fs.Set(key2, time.Now().Add(20*time.Minute).Unix(), data2)
	if err != nil {
		fmt.Errorf("Failed set data 2 err=%v", err)
	}
	fs.Set(key3, time.Now().Add(20*time.Minute).Unix(), data3)
	if err != nil {
		fmt.Errorf("Failed set data 3 err=%v", err)
	}

	err = fs.Delete(key3)
	if err != nil {
		fmt.Errorf("Failed delete key 3 err=%v", err)
	}

	err = fs.Delete(key3)
	if err == nil {
		fmt.Errorf("Failed delete dels key 3, key 3 already delete")
	}

	data1_get, err := fs.Get(key1)
	if err != nil {
		fmt.Errorf("Failed get data 1 string err=%v", err)
	}

	if string(data1_get) != "my data 1" {
		fmt.Errorf("Failed get data 1 get string dont equal set val")
	} 

	data2_get, err := fs.Get(key2)
	if err != nil {
		fmt.Errorf("Failed get data 1 string err=%v", err)
	}

	if !slices.Equal(data2, data2_get) {
		fmt.Errorf("Failed get data 2 slices don't equals")
	}

	data3_get, err := fs.Get(key3)
	if err != nil {
		fmt.Errorf("Failed get data 1 string err=%v", err)
	}

	if data3_get != nil {
		fmt.Errorf("Failed get data 3 this is del elem")
	}

	fs.Close()
}

func TestFileShardConcurencyBench(t *testing.T) {
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel() 

	worker := 2
	var qt_query atomic.Int32
	var all_time atomic.Int64
	chance_set := 90

	keys := []string{"key1", "key2", "key3", "key4", "key5", "key6"}
	tmp := `C:\Temp\proxy`
	fs, err := stores.NewFileShard(&tmp, 1*stores.Mbyte)
	if err != nil {
		fmt.Errorf("Failed new file shard err=%v", err)
	}

	for _, k := range keys {
		fs.Set(k, time.Now().Add(20*time.Minute).Unix(), []byte(k))
	}

	var wg sync.WaitGroup

	for range worker {
		wg.Go(func() {

			for {
				select {
				case <-ctx.Done():
					return
				default:
					isSet := rand.IntN(100) > chance_set
					keyName := fmt.Sprintf("key%v", rand.IntN(10))
					data := fmt.Sprintf("data%v", rand.IntN(100))
					if isSet {
						
						start := time.Now()

						err = fs.Set(keyName, time.Now().Add(20*time.Minute).Unix(), []byte(data))
						if err != nil {
							fmt.Errorf("failed set err=%v key=%v", err, keyName)
						}				

						all_time.Add(time.Since(start).Microseconds())

						qt_query.Add(1)

					} else {
						
						start := time.Now()

						_, err := fs.Get(keyName)
						if err != nil {
							fmt.Errorf("failed get err=%v key=%v", err, keyName)
						}

						all_time.Add(int64((time.Since(start).Seconds())))

						// if val != nil {
						// 	fmt.Print(string(val) + "\n")
						// } else {
						// 	fmt.Print("nil\n")
						// }
						
						qt_query.Add(1)
					}
				}
			}
		})
	}
	wg.Wait()

	all_qr := qt_query.Load()
	all_time_req := all_time.Load()
	fmt.Printf("all query - %v\nmiddle time - %v\n", all_qr, all_time_req/int64(all_qr))
}

func TestConvertRecordToBiteSlice(t *testing.T) {
	
	key := "my_key"
	value := "my_value_is_a_big_value"

	rec, size := stores.NewRecord(key, []byte(value))

	byteSlice := rec.ToByte()

	if size != int64(len(byteSlice)) {
		t.Errorf("failed convert record to byte slice size=%v len(byteSlice)=%v", size, int64(len(byteSlice)))
	}

}

func BenchmarkConvertRecordToBiteSliceUseAppendDontDefineSize(b *testing.B) {
	
	key := "my_key"
	value := "my_value_is_a_big_value"

	rec, _ := stores.NewRecord(key, []byte(value))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {	
		_ = rec.ToByte()
	}

}