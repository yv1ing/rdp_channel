package transport

type Transport interface {
	Read() (int, []byte, error)
	Write([]byte) (int, error)
}
