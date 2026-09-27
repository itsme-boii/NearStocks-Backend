package ctypes

type RedisField interface {
	UnmarshalBinary(data []byte) error
	MarshalBinary() (data []byte, err error)
}
