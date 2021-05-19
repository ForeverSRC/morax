package consumer

type ConsumerConfig struct {
	Reference ReferenceConfig `mapstructure:"reference"`
}

type ReferenceConfig struct {
	ConfInfo  `mapstructure:",squash"`
	Providers map[string]ProviderServiceConfig `mapstructure:"providers"`
}

type ProviderServiceConfig struct {
	ConfInfo `mapstructure:",squash"`
	Methods  map[string]MethodConfig `mapstructure:"methods"`
}

type MethodConfig struct {
	ConfInfo `mapstructure:",squash"`
}

type ConfInfo struct {
	LBType  string `mapstructure:"loadBalance"`
	Retries int    `mapstructure:"retries"`
	Timeout int    `mapstructure:"timeout"`
	Cluster string `mapstructure:"cluster"`
}
