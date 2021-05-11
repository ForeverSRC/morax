package loadbalance

import (
	"errors"
	"github.com/ForeverSRC/morax/common/constants"
	"github.com/ForeverSRC/morax/registry/consul"
)

type RoundRobin struct {
	curIdx int
}

func init() {
	RegisterBalance(constants.RoundRobin, &RoundRobin{})
}

func (r *RoundRobin) DoBalance(instances []*consul.ServiceInstance) (*consul.ServiceInstance, error) {
	lens := len(instances)
	if lens == 0 {
		return nil, errors.New("no instance found")
	}

	if r.curIdx >= lens {
		r.curIdx = 0
	}

	inst := instances[r.curIdx]
	r.curIdx = (r.curIdx + 1) % lens

	return inst, nil
}
