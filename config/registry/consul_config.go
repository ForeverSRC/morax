package registry

type ConsulClientConfig struct {
	Addr        string `mapstructure:"addr"`
	WaitTimeout int    `mapstructure:"waitTimeout"`
}
