package loadbalance

import (
	"errors"
	"math/rand"
	"time"
)

import (
	"github.com/ForeverSRC/morax/common/constants"
)

type ShuffleBalance struct {
}

func init() {
	RegisterBalance(constants.ShuffleBalance, &ShuffleBalance{})
}

func (s *ShuffleBalance) DoBalance(instanceIds []string) (string, error) {
	lens := len(instanceIds)
	if lens == 0 {
		return "", errors.New("no instance found")
	}

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < lens/2; i++ {
		a := rand.Intn(lens)
		b := rand.Intn(lens)
		instanceIds[a], instanceIds[b] = instanceIds[b], instanceIds[a]
	}

	inst := instanceIds[0]
	return inst, nil
}
