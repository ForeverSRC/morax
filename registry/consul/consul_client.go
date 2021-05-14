package consul

import (
	"fmt"
	"time"
)

import (
	cp "github.com/ForeverSRC/morax/config/provider"
	cr "github.com/ForeverSRC/morax/config/registry"
	"github.com/ForeverSRC/morax/logger"
)

import (
	consulapi "github.com/hashicorp/consul/api"
)

var client *consulapi.Client

func NewClient(dcf *cr.ConsulClientConfig) {
	var err error
	conf := consulapi.DefaultConfig()
	conf.Address = dcf.Addr
	if dcf.WaitTimeout == 0 {
		conf.WaitTime = 5 * time.Minute
	} else {
		conf.WaitTime = time.Duration(dcf.WaitTimeout) * time.Second
	}
	client, err = consulapi.NewClient(conf)
	if err != nil {
		logger.Fatal(err)
	}
}

func Register(info *cp.ProviderConfig) error {
	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = info.GenerateProviderID()
	registration.Name = info.Service.Name
	registration.Port = info.Service.Port
	registration.Address = info.Service.Host

	check := new(consulapi.AgentServiceCheck)
	check.TCP = fmt.Sprintf("%s:%d", registration.Address, info.Service.Check.CheckPort)
	check.Timeout = info.Service.Check.Timeout
	check.Interval = info.Service.Check.Interval
	check.DeregisterCriticalServiceAfter = info.Service.Check.DeregisterAfter // 故障检查失败30s后 consul自动将注册服务删除
	registration.Check = check

	err := client.Agent().ServiceRegister(registration)

	if err != nil {
		logger.Error("register error: %s", err)
		return err
	}

	logger.Info("register service success!")
	return nil

}

func FindServers(name string, idx uint64) ([]*consulapi.ServiceEntry, *consulapi.QueryMeta, error) {
	// 阻塞
	return client.Health().Service(name, "", true, &consulapi.QueryOptions{WaitIndex: idx})
}
