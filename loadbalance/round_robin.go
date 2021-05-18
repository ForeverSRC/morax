package loadbalance

import (
	"errors"
	"github.com/ForeverSRC/morax/common/constants"
)

type RoundRobin struct {
	curIdx int
}

func init() {
	RegisterBalance(constants.RoundRobin, &RoundRobin{})
}

func (r *RoundRobin) DoBalance(instanceIds []string) (string, error) {
	lens := len(instanceIds)
	if lens == 0 {
		return "", errors.New("no instance found")
	}

	if r.curIdx >= lens {
		r.curIdx = 0
	}

	inst := instanceIds[r.curIdx]
	r.curIdx = (r.curIdx + 1) % lens

	return inst, nil
}
