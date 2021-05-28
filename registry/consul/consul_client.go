package consul

import (
	"context"
	"net/http"
	"time"
)

import (
	cr "github.com/ForeverSRC/morax/config/registry"
	"github.com/ForeverSRC/morax/logger"
)

import (
	consulapi "github.com/hashicorp/consul/api"
)

type consulClientInfo struct {
	consulClient *consulapi.Client
	httpClient   *http.Client
}

var clientInfo *consulClientInfo

func NewClient(dcf *cr.ConsulClientConfig) {
	var err error
	conf := consulapi.DefaultConfig()
	conf.Address = dcf.Addr
	if dcf.WaitTimeout == 0 {
		conf.WaitTime = 5 * time.Minute
	} else {
		conf.WaitTime = time.Duration(dcf.WaitTimeout) * time.Second
	}

	client, err := consulapi.NewClient(conf)
	if err != nil {
		logger.Fatal(err)
	}
	clientInfo = &consulClientInfo{consulClient: client, httpClient: conf.HttpClient}
}

func Register(registration *consulapi.AgentServiceRegistration) error {
	err := clientInfo.consulClient.Agent().ServiceRegister(registration)

	if err != nil {
		logger.Error("register error: %s", err)
		return err
	}

	logger.Info("register service success!")
	return nil
}

func FindServers(ctx context.Context, name string, idx uint64) ([]*consulapi.ServiceEntry, *consulapi.QueryMeta, error) {
	qo := &consulapi.QueryOptions{WaitIndex: idx}
	qo = qo.WithContext(ctx)
	// 阻塞
	return clientInfo.consulClient.Health().Service(name, "", true, qo)
}

func Deregister(id string) error {
	return clientInfo.consulClient.Agent().ServiceDeregister(id)
}

func CloseIdleConn() {
	clientInfo.httpClient.CloseIdleConnections()
}
