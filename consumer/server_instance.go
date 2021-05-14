package consumer

import (
	"github.com/ForeverSRC/morax/logger"
	"github.com/ForeverSRC/morax/registry/consul"
	"sync"
)

type ConsumeServersStore struct {
	providerName string
	m            []*consul.ServiceInstance
	idx          uint64
	mu           sync.RWMutex
}

func NewConsumeServersStore(name string) *ConsumeServersStore {
	return &ConsumeServersStore{
		providerName: name,
	}
}
func (cs *ConsumeServersStore) Get() []*consul.ServiceInstance {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.m
}

func (cs *ConsumeServersStore) Set(insts []*consul.ServiceInstance) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.m = insts
}

func (cs *ConsumeServersStore) SetWithIndex(insts []*consul.ServiceInstance, idx uint64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.m = insts
	cs.idx = idx
}

func (cs *ConsumeServersStore) Watch() bool {
	logger.Debug("Servers find start")
	// 阻塞
	services, meta, err := consul.FindServers(cs.providerName, cs.idx)
	logger.Debug("Servers find return")

	if err != nil {
		logger.Error("find provider %s error:%s", cs.providerName, err)
		cs.Set(nil)
		return false
	}

	if len(services) == 0 {
		logger.Warn("find service: %s instance zero!", cs.providerName)
		cs.Set(nil)
		return false
	}

	instances := make([]*consul.ServiceInstance, len(services))
	for i, s := range services {
		instances[i] = &consul.ServiceInstance{
			Address: s.Service.Address,
			Port:    s.Service.Port,
		}
	}

	logger.Debug("Servers find return，pre index:%d, return index:%d", cs.idx, meta.LastIndex)
	if meta.LastIndex < cs.idx {
		cs.SetWithIndex(instances, 0)
	} else {
		cs.SetWithIndex(instances, meta.LastIndex+1)
	}

	return true
}
