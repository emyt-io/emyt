package config

type Config struct {
	Version         string    `yaml:"version"`
	StatusHost      string    `yaml:"status_host"`
	ProxyListenPort string    `yaml:"proxy_listen_port"`
	Logfile         string    `yaml:"log_file"`
	Services        []Service `yaml:"services"`
}

type Service struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"` // Type is one of ['web', 'proxy', 'static']
	IngressUrl string `yaml:"ingress_url"`
	EgressUrl  string `yaml:"egress_url"`
	UseAuth    bool   `yaml:"use_auth"`
}
