package dns

type Config struct {
	DefaultUpstream string   `yaml:"upstream_server"`
	ListenPort      int8     `yaml:"listen"`
	ConfigFiles     []string `yaml:"config_files"`
}
