package loadbalance

import (
	"errors"
	"github.com/morax/common/constants"
	"github.com/morax/registry/consul"
	"math/rand"
	"time"
)

type ShuffleBalance struct {
}

func init() {
	RegisterBalance(constants.ShuffleBalance, &ShuffleBalance{})
}

func (s *ShuffleBalance) DoBalance(instances []*consul.ServiceInstance) (*consul.ServiceInstance, error) {
	lens := len(instances)
	if lens == 0 {
		return nil, errors.New("no instance found")
	}

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < lens/2; i++ {
		a := rand.Intn(lens)
		b := rand.Intn(lens)
		instances[a], instances[b] = instances[b], instances[a]
	}

	inst := instances[0]
	return inst, nil
}
