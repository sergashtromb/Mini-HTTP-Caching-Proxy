package main

import (
	//"fmt"
	"context"
	"fmt"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/initial/stores"
	"mini_http_caching_proxy/logger"
	"time"
)

type MyStruct struct {
	pol int16
	str string
}

func main() {

	cnf := config.Init("config.yaml", false)

	logFile, err := logger.Init(cnf.LogSettings.Directory, cnf.LogSettings.Level)
	if err != nil {
		fmt.Println("Error load logger err:", err)
		return
	}
	defer logFile.Close()

	cachStore := stores.NewRamCacheStore(&cnf, 5*time.Second, 10)

 
	bytesType := make([]byte, 10)
	strType := "myString"
	intType := 150
	ms1 := MyStruct {
		pol: 52,
		str: "My struct 1",
	}

	msForPointer := &MyStruct {
		pol: 568,
		str: "My Pointer struct ",
	}

	boolType := false

	cachStore.Set("randomKeyBytesType", bytesType, 10* time.Second)
	cachStore.Set("randomKeyStrType", strType, 10*time.Second)
	cachStore.Set("randomKeyIntType", intType, 10*time.Second)
	cachStore.Set("randomKeyBoolType", boolType, 10*time.Second)
	cachStore.Set("randomKeyms1", ms1, 10*time.Second)
	cachStore.Set("randomKeymsForPointer", msForPointer, 10*time.Second)

	bytesTypeGetVar := cachStore.Get("randomKeyBytesType")
	strTypeGetVar := cachStore.Get("randomKeyStrType")
	intTypeGetVar := cachStore.Get("randomKeyIntType")
	boolTypeGetVar := cachStore.Get("randomKeyBoolType")
	msTypeGetVar := cachStore.Get("randomKeyms1")
	pointerTypeGetVar := cachStore.Get("randomKeymsForPointer")

	cachStore.DelExpiration(context.Background())

	fmt.Printf("byte - %v\n", bytesTypeGetVar.([]byte))
	fmt.Printf("string - %v\n", strTypeGetVar.(string))
	fmt.Printf("int - %v\n", intTypeGetVar.(int))
	fmt.Printf("bool - %v\n", boolTypeGetVar.(bool))
	fmt.Printf("ms - %v\n", msTypeGetVar.(MyStruct))
	fmt.Printf("pointer - %v\n", pointerTypeGetVar.(*MyStruct))
	time.Sleep(6*time.Second)

	bt2 := cachStore.Get("randomKeyBytesType")

	fmt.Printf("byte - %v\n", bt2)

}