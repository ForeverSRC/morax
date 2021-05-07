package consul

type InstanceConfig struct {
	ServiceName string `json:"serviceName"`
	Address string `json:"address"`
	Port string `json:"port"`
}

type DiscoveryClientConfig struct {
	Host string
	Port int
}



