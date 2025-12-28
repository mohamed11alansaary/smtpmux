package types

type Downstream struct {
	Addr string `yaml:"addr"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}
type UserConfig struct {
	Email            string       `yaml:"email"`
	Password         string       `yaml:"password"`
	SelectorAlgoPath string       `yaml:"selector_algo_path"`
	Downstreams      []Downstream `yaml:"downstreams"`
}

type Config struct {
	Users []UserConfig `yaml:"users"`
}
