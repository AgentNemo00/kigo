package frame

type Frame struct {
	read func() ([]byte, error)
	close func()
	name func() string
}

func (f *Frame) Read() ([]byte, error) {
	return f.read()
}

func (f *Frame) Close() {
	f.close()
}

func (f* Frame) Name() string {
	return f.name()
}