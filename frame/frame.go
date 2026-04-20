package frame

type Frame struct {
	read func() ([]byte, error)
	close func()
}

func (f *Frame) Read() ([]byte, error) {
	return f.read()
}

func (f *Frame) Close() {
	f.close()
}
