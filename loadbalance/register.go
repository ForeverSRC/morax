package loadbalance

import (
	"fmt"
	"github.com/morax/registry/consul"
)

type Balances struct {
	allBalance map[string]Balance
}

var balances = Balances{
	allBalance: make(map[string]Balance),
}

func (bs *Balances) RegisterBalance(balanceType string, b Balance) {
	bs.allBalance[balanceType] = b
}

func RegisterBalance(balanceType string, b Balance) {
	balances.allBalance[balanceType] = b
}

func DoBalance(bType string, instances []*consul.ServiceInstance) (*consul.ServiceInstance, error) {
	balanceType, ok := balances.allBalance[bType]
	if !ok {
		return nil, fmt.Errorf("un found balance type:%s\n", bType)
	}

	return balanceType.DoBalance(instances)
}
