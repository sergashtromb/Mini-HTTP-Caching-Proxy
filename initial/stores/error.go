package stores

import "fmt"

type StoreError struct {
	StCode string
	Message string
}

func (st *StoreError) Error() string {
	return fmt.Sprintf("StoreError :%s: %s", st.StCode, st.Message)
}

func LossOfRecording(mes string) error {
	return &StoreError {
		StCode: "LOSS_OF_RECORDING",
		Message: mes,
	}
}

func MemoryLimit(mes string) error {
	return &StoreError{
		StCode: "MEMORY_LIMIT",
		Message: mes,
	}
}

func FailedRecordToCache(mes string) error {
	return &StoreError{
		StCode: "FAILED_RECORD_TO_CACHE",
		Message: mes,
	}
}

func FailedDeleteRecordToCache(mes string) error {
	return &StoreError{
		StCode: "FAILED_DELETE_RECORD_TO_CACHE",
		Message: mes,
	}
}