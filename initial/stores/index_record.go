package stores

import "time"

type IndexRecord struct {
	Key			[]byte
	offset 		int64
	Expiration 	int64
	KeyLen 		uint32
}

func NewIndexRecord(key string, offset, expiration int64) IndexRecord {

	key_byte := []byte(key)
	key_len := uint32(len(key_byte))

	return IndexRecord{
		Key: key_byte,
		offset: offset,
		Expiration: expiration,
		KeyLen: key_len,
	}
}

func NewIndexRecordFromRecord(key string, data *Record, fl_size, expiration int64) (IndexRecord, int64) {

	ir := NewIndexRecord(key, fl_size, expiration)
	size := fl_size + data.RecSize()

	return ir, size
}

func (ir *IndexRecord) IsDead() bool {

	now := time.Now()
	d := now.Sub(time.Unix(ir.Expiration, 0)).Seconds()

	return d > 0.0
}

func (ir *IndexRecord) GetKey() string {
	return string(ir.Key)
}

func (ir *IndexRecord) GetOffset() int64 {
	return ir.offset
}