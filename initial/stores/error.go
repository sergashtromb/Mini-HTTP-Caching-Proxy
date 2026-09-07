package stores

import "fmt"

type StoreError struct {
	StCode string
	Message string
}

func (st *StoreError) Error() string {
	return fmt.Sprintf("StoreError :%s: %s\n", st.StCode, st.Message)
}

func LossOfRecording(mes string) error {
	return &StoreError {
		StCode: "LOSS_OF_RECORDING",
		Message: mes,
	}
}