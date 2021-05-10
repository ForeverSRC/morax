package loadbalance

import (
	"errors"
	"github.com/morax/common/constants"
	"github.com/morax/registry/consul"
	"math/rand"
)

type RandomBalance struct {
}

func init() {
	RegisterBalance(constants.RandomBalance, &RandomBalance{})
}

func (r *RandomBalance) DoBalance(instances []*consul.ServiceInstance) (*consul.ServiceInstance, error) {
	lens := len(instances)
	if lens == 0 {
		return nil, errors.New("no instance found")
	}

	index := rand.Intn(lens)
	inst := instances[index]
	return inst, nil
}
