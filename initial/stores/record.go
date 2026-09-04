package stores


type Record struct {
	Delete	byte
	Key 	[]byte
	KeyLen	uint32 // 4 size
	Object 	[]byte
	ObjLen 	uint32 // 4 size
}

func NewRecord(key string, data []byte) (Record, int64) {

	key_byte := []byte(key)
	key_len := uint32(len(key_byte))
	obj_len := uint32(len(data))

	rec := Record {
		Key: key_byte,
		Object: data,
		KeyLen: key_len,
		ObjLen: obj_len,
	}
	size := rec.RecSize()

	return rec, size
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

func (r *Record) IsDel() bool {
	return r.Delete == 0
}