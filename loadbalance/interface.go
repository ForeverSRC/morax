package loadbalance

import "github.com/morax/registry/consul"

type Balance interface {
	DoBalance([]*consul.ServiceInstance) (*consul.ServiceInstance, error)
}
