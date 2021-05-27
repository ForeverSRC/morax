package service

type ServiceConfig struct {
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
}
