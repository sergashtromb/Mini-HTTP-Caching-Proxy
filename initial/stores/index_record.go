package stores

import (
	"time"
)

type IndexRecord struct {
	offset 		int64
	Expiration 	int64
	DateLen		int64
}

func NewIndexRecord(offset, expiration, dateLen int64) IndexRecord {
	return IndexRecord{
		offset: offset,
		Expiration: expiration,
		DateLen: dateLen,
	}
}

func NewIndexRecordFromRecord(data *Record, fl_size, expiration int64) (IndexRecord, int64) {

	size := data.RecSize()
	ir := NewIndexRecord(fl_size, expiration, size)
	new_fl_size := fl_size + size

	return ir, new_fl_size
}

func (ir *IndexRecord) IsDead() bool {

	now := time.Now()
	d := now.Sub(time.Unix(ir.Expiration, 0)).Seconds()

	return d > 0.0
}

func (ir *IndexRecord) IsDeadFromTime(tm time.Time) bool {
	
	d := tm.Sub(time.Unix(ir.Expiration, 0)).Seconds()
	return d > 0.0
}

func (ir *IndexRecord) GetOffset() int64 {
	return ir.offset
}