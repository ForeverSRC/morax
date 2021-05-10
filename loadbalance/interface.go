package loadbalance

import "github.com/ForeverSRC/morax/registry/consul"

type Balance interface {
	DoBalance([]*consul.ServiceInstance) (*consul.ServiceInstance, error)
}
