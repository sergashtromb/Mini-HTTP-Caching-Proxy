package test

import (

	"mini_http_caching_proxy/initial/stores"
	"testing"
	"time"
)

func TestCheckDataToDel(t *testing.T) {

	time_plus_11S := time.Now().Add(10*time.Second)

	timeToDel := time.Now().Unix()

	ir := stores.NewIndexRecord(10, timeToDel, 10)

	if !ir.IsDeadFromTime(time_plus_11S) {
		t.Errorf("Failed check index record to dead")
	}

	
	if ir.IsDeadFromTime(time.Now()) {
		t.Errorf("Failed check index record to dead")
	}

}