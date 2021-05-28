package provider

type ServiceConfig struct {
	Port int `mapstructure:"port"`
}

type ProviderConfig struct {
	Service ServiceConfig `mapstructure:"service"`
}
