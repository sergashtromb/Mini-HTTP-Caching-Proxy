package stores

import "fmt"

type ErrorCode string

const (
	LossOfRecordingCode 			ErrorCode = "LOSS_OF_RECORDING"
	MemoryLimitCode 				ErrorCode = "MEMORY_LIMIT"
	FailedRecordToCacheCode 		ErrorCode = "FAILED_RECORD_TO_CACHE"
	FailedDeleteRecordToCacheCode 	ErrorCode = "FAILED_DELETE_RECORD_TO_CACHE"
	InteruptedCompactizationCode	ErrorCode = "INTERUPTED_COMPACTIZATION"
)

var (
	ErrLossOfRecording 				= &StoreError{StCode: LossOfRecordingCode}
	ErrMemoryLimit 					= &StoreError{StCode: MemoryLimitCode}
	ErrFailedRecordToCache 			= &StoreError{StCode: FailedRecordToCacheCode}
	ErrFailedDeleteRecordToCache 	= &StoreError{StCode: FailedDeleteRecordToCacheCode}
	ErrInteruptedCompactization		= &StoreError{StCode: InteruptedCompactizationCode}
)

type StoreError struct {
	StCode ErrorCode
	Message string
}

func (st *StoreError) Error() string {
	return fmt.Sprintf("StoreError :%s: %s", st.StCode, st.Message)
}

func LossOfRecording(mes string) error {
	return fmt.Errorf("%w: %s", ErrLossOfRecording, mes)
}

func MemoryLimit(mes string) error {
	return fmt.Errorf("%w: %s", ErrMemoryLimit, mes)
}

func FailedRecordToCache(mes string) error {
	return fmt.Errorf("%w: %s", ErrFailedRecordToCache, mes)
}

func FailedDeleteRecordToCache(mes string) error {
	return fmt.Errorf("%w: %s", ErrFailedDeleteRecordToCache, mes)
}

func InteruptedCompactization() error {
	return fmt.Errorf("%w", ErrInteruptedCompactization)
}