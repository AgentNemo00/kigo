package core

type Config struct {
	Port int
	Modules []Module
}

func (mc *Config) Default() {
	if mc.Port == 0 {
		mc.Port = 10001
	}
}

type Module struct {
	Name string
	Path string
}
