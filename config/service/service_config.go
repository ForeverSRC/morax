package service

import "fmt"

type ServiceConfig struct {
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
}

func (sc *ServiceConfig) GenerateID() string {
	return fmt.Sprintf("%s-%s", sc.Name, sc.Host)
}
