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
	mu           sync.Mutex
}

func NewConsumeServersStore(name string) *ConsumeServersStore {
	return &ConsumeServersStore{
		providerName: name,
	}
}
func (cs *ConsumeServersStore) Get() []*consul.ServiceInstance {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.m
}

func (cs *ConsumeServersStore) Set(insts []*consul.ServiceInstance) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.m = insts
}

func (cs *ConsumeServersStore) Watch() {
	logger.Debug("Servers find start")
	// 阻塞
	services, meta, err := consul.FindServers(cs.providerName, cs.idx)
	logger.Debug("Servers find return")

	if err != nil {
		logger.Error("find provider %s error:%s", cs.providerName, err)
		cs.mu.Lock()
		cs.m = nil
		cs.mu.Unlock()
		return
	}

	if len(services) == 0 {
		cs.mu.Lock()
		cs.m = nil
		cs.mu.Unlock()
		return
	}

	instances := make([]*consul.ServiceInstance, len(services))
	for i, s := range services {
		instances[i] = &consul.ServiceInstance{
			Address: s.Service.Address,
			Port:    s.Service.Port,
		}
	}

	cs.mu.Lock()
	cs.m = instances
	if meta.LastIndex < cs.idx {
		cs.idx = 0
	} else {
		cs.idx = meta.LastIndex + 1
	}
	cs.mu.Unlock()

}
