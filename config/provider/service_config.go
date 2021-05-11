package provider

import "fmt"

type ServiceConfig struct {
	Name  string             `mapstructure:"name"`
	Host  string             `mapstructure:"host"`
	Port  int                `mapstructure:"port"`
	Check ServiceCheckConfig `mapstructure:"check"`
}

type ProviderConfig struct {
	Service ServiceConfig `mapstructure:"service"`
}

type ServiceCheckConfig struct {
	CheckPort       int    `mapstructre:"CheckPort"`
	CheckUri        string `mapstructure:"checkUri"`
	Timeout         string `mapstructure:"timeout"`
	Interval        string `mapstructure:"interval"`
	DeregisterAfter string `mapstructure:"deregisterAfter"`
}

func (p *ProviderConfig) GenerateProviderID() string {
	return fmt.Sprintf("%s-%s:%d", p.Service.Name, p.Service.Host, p.Service.Port)
}
