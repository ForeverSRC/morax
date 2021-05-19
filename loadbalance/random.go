package loadbalance

import (
	"errors"

	"math/rand"
)

import (
	"github.com/ForeverSRC/morax/common/constants"
)

type RandomBalance struct {
}

func init() {
	RegisterBalance(constants.RandomBalance, &RandomBalance{})
}

func (r *RandomBalance) DoBalance(instanceIds []string) (string, error) {
	lens := len(instanceIds)
	if lens == 0 {
		return "", errors.New("no instance found")
	}

	index := rand.Intn(lens)
	inst := instanceIds[index]
	return inst, nil
}
