package loadbalance

import (
	"errors"

	"math/rand"
)

import (
	"github.com/ForeverSRC/morax/common/constants"
	"github.com/ForeverSRC/morax/registry/consul"
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
