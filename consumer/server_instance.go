package consumer

import (
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sort"
	"sync"
)

import (
	"github.com/ForeverSRC/morax/loadbalance"
	"github.com/ForeverSRC/morax/logger"
	"github.com/ForeverSRC/morax/registry/consul"
)

type providerInstance struct {
	id   string
	host string
	port int
}

// ProviderInstances 提供者集群信息
type ProviderInstances struct {
	// providerName 订阅的服务名
	providerName string
	// instances provider实例map ID->rpc.Client
	instances map[string]*rpc.Client
	ids       []string
	idx       uint64
	mu        sync.RWMutex
}

func NewProviderInstances(name string) *ProviderInstances {
	return &ProviderInstances{
		providerName: name,
		instances:    make(map[string]*rpc.Client),
	}
}

func (ps *ProviderInstances) LoadBalance(lbType string) (*rpc.Client, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if ps.instances == nil {
		return nil, fmt.Errorf("provider: %s zero instance", ps.providerName)
	}
	inst, err := loadbalance.DoBalance(lbType, ps.ids)
	if err != nil {
		return nil, err
	}

	return ps.instances[inst], nil
}

func (ps *ProviderInstances) setLocked(key string, value *providerInstance) {
	target := fmt.Sprintf("%s:%d", value.host, value.port)
	client, err := jsonrpc.Dial("tcp", target)
	if err != nil {
		logger.Error("connect to %s error: %s", target, err)
		return
	}

	ps.instances[key] = client
}

func (ps *ProviderInstances) setIndexLocked(idx uint64, setZero bool) {
	if setZero {
		ps.idx = 0
	} else {
		ps.idx = idx
	}
}

func (ps *ProviderInstances) setInstancesIds() {
	count := len(ps.instances)
	ids := make([]string, count)
	i := 0
	for k := range ps.instances {
		ids[i] = k
	}

	sort.Strings(ids)
	ps.ids = ids
}

func (ps *ProviderInstances) Watch() <-chan bool {
	logger.Debug("Servers find start")
	// 阻塞
	services, meta, err := consul.FindServers(ps.providerName, ps.idx)
	logger.Debug("Servers find return")

	resCh := make(chan bool)
	defer close(resCh)
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if err != nil {
		logger.Error("find provider %s error:%s", ps.providerName, err)
		ps.instances = nil
		resCh <- false
		return resCh
	}

	if len(services) == 0 {
		logger.Warn("find service: %s instance zero!", ps.providerName)
		ps.instances = nil
		resCh <- false
		return resCh
	}

	mp := make(map[string]*providerInstance)
	if ps.instances == nil {
		ps.instances = make(map[string]*rpc.Client)
	}

	for _, s := range services {
		i := &providerInstance{
			id:   s.Service.ID,
			host: s.Service.Address,
			port: s.Service.Port,
		}
		mp[s.Service.ID] = i

		// 之前不存在而现在存在的实例进行新增
		if _, ok := ps.instances[s.Service.ID]; !ok {
			ps.setLocked(s.Service.ID, i)
		}
	}

	ps.setInstancesIds()

	for k, v := range ps.instances {
		// 之前存在现在不存在的要剔除
		if _, ok := mp[k]; !ok {
			_ = v.Close()
			delete(ps.instances, k)
		}
		// 之前存在现在也存在的实例不变
	}

	ps.setIndexLocked(meta.LastIndex, meta.LastIndex < ps.idx)

	resCh <- true
	return resCh
}
