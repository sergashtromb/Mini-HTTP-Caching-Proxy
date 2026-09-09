package stores

import (
	"encoding/binary"
	//"fmt"
)


type Record struct {
	Delete	byte
	KeyLen	uint32 // 4 size
	Key 	[]byte
	ObjLen 	uint32 // 4 size
	Object 	[]byte
}

func NewRecord(key string, data []byte) (Record, int64) {

	key_byte := []byte(key)
	key_len := uint32(len(key_byte))
	obj_len := uint32(len(data))

	rec := Record {
		Delete: 0,
		Key: key_byte,
		Object: data,
		KeyLen: key_len,
		ObjLen: obj_len,
	}
	size := rec.RecSize()

	return rec, size
}

func NewRecordFromParams(key_ln, obj_ln uint32, key_bt, val_bt []byte) Record {
	return Record {
		Delete: 0,
		Key: key_bt,
		Object: val_bt,
		KeyLen: key_ln,
		ObjLen: obj_ln,
	}
}

func (r *Record) RecSize() int64 {
	return int64(4 + len(r.Key) + 4 + len(r.Object) + 1)
}

func (r *Record) GetKey() string {
	return string(r.Key)
}

func (r *Record) GetData() *[]byte {
	return &r.Object
}

func (r *Record) ToByte() []byte {

	rec_bt_slice := make([]byte, r.RecSize())
	binary.BigEndian.PutUint32(rec_bt_slice[1:5], r.KeyLen)

	offset := 5
	offset += copy(rec_bt_slice[offset:], r.Key)

	binary.BigEndian.PutUint32(rec_bt_slice[offset:offset+4], r.ObjLen)
	offset += 4

	copy(rec_bt_slice[offset:], r.Object)
	
	return rec_bt_slice
}

func RecordFromBytesSlice(b []byte) Record {

	rec := Record {}
	var offset int64
	rec.Delete = b[0]
	rec.KeyLen = binary.BigEndian.Uint32(b[1:5])

	//fmt.Printf("\n%v\n\t\t%v %v\n\n", b, b[1:5], binary.BigEndian.Uint32(b[1:5]))

	offset = 5
	rec.Key = b[offset:offset+int64(rec.KeyLen)]

	offset += int64(rec.KeyLen)
	rec.ObjLen = binary.BigEndian.Uint32(b[offset:offset+4])

	rec.Object = b[offset+4:]

	return rec
}