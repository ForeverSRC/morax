package service

import (
	"context"
	"fmt"
	"time"
)

import (
	"github.com/ForeverSRC/morax/common/utils"
	ck "github.com/ForeverSRC/morax/config/check"
	cc "github.com/ForeverSRC/morax/config/consumer"
	cp "github.com/ForeverSRC/morax/config/provider"
	cs "github.com/ForeverSRC/morax/config/service"
	"github.com/ForeverSRC/morax/consumer"
	"github.com/ForeverSRC/morax/provider"
	"github.com/ForeverSRC/morax/registry/consul"
)

import (
	consulapi "github.com/hashicorp/consul/api"
)

const DEFAULT_SERVICE_PORT = 8888

type MoraxService struct {
	name    string
	host    string
	id      string
	rpcPort int
	ctx     context.Context
	pro     *provider.RpcProvider
	con     *consumer.RpcConsumer
	check   *consul.HealthCheckService
}

// 初始化API 可以通过config包从配置文件中读取配置，也可自定义配置类
// InitService 初始化服务信息
func (ms *MoraxService) InitService(sf *cs.ServiceConfig, ctx context.Context) error {
	ms.name = sf.Name
	if sf.Host == "" {
		address, err := utils.GetLocalAddr()
		if err != nil {
			return err
		}

		ms.host = address
	} else {
		ms.host = sf.Host
	}
	ms.rpcPort = DEFAULT_SERVICE_PORT
	ms.ctx = ctx
	return nil
}

func (ms *MoraxService) generateId() string {
	ms.id = fmt.Sprintf("%s-%s:%d", ms.name, ms.host, ms.rpcPort)
	return ms.id
}

// InitHealthCheck 初始化健康检查服务
func (ms *MoraxService) InitHealthCheck(ckf *ck.CheckConfig) {
	healthCheck := consul.NewHealthCheckService(ms.host, ckf)
	ms.check = healthCheck
}

// InitRpcConsumer 初始化rpc consumer
func (ms *MoraxService) InitRpcConsumer(cmf *cc.ConsumerConfig) {
	con := consumer.NewRpcConsumer(ms.ctx, cmf)
	ms.con = con
}

// InitRpcProvider 初始化rpc provider
func (ms *MoraxService) InitRpcProvider(pvf *cp.ProviderConfig) {
	ms.rpcPort = pvf.Service.Port
	pro := provider.NewRpcProvider(ms.host, pvf)
	ms.pro = pro
}

// 注册API
// RegisterProvider 注册提供的方法
func (ms *MoraxService) RegisterProvider(methods interface{}) error {
	if ms.name == "" {
		return fmt.Errorf("service name is blank")
	}

	if ms.pro == nil {
		return fmt.Errorf("provider is not initialized")
	}

	return ms.pro.RegisterProvider(ms.name, methods)
}

// RegisterConsumer 注册消费的方法
func (ms *MoraxService) RegisterConsumer(name string, service interface{}) error {
	if ms.con == nil {
		return fmt.Errorf("consumer is not initialized")
	}

	return ms.con.RegisterConsumer(name, service)
}

// ListenAndServe 启动服务
func (ms *MoraxService) ListenAndServe() error {
	// 注册服务
	registration := ms.genRegistration()
	err := consul.Register(registration)
	if err != nil {
		return err
	}

	// 启动健康检查服务
	if ms.check == nil {
		return fmt.Errorf("health check service is not initialized")
	}
	ms.check.ListenAndServe()

	// 启动provider（如果有）
	if ms.pro != nil {
		ms.pro.ListenAndServe()
	}
	return nil
}

func (ms *MoraxService) genRegistration() *consulapi.AgentServiceRegistration {
	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = ms.generateId()
	registration.Name = ms.name
	registration.Port = ms.rpcPort
	registration.Address = ms.host
	registration.Check = ms.check.CheckInfo
	return registration
}

func (ms *MoraxService) Shutdown(ctx context.Context) error {
	// 向注册中心注销实例
	_ = consul.Deregister(ms.id)

	// 健康检查关机
	_ = ms.check.Shutdown()

	// 消费者优雅关机（如果有）
	if ms.con != nil {
		ms.con.Shutdown()
	}

	// 提供者优雅关机（如果有）
	if ms.pro != nil {
		_ = ms.pro.Shutdown()
	}
	// 关闭consul client idle connections
	consul.CloseIdleConn()

	pollIntervalBase := time.Millisecond
	timer := time.NewTimer(utils.NextPollInterval(&pollIntervalBase))
	defer timer.Stop()
	for {
		if ms.pro == nil {
			return nil
		} else if ms.pro.CloseIdleCodecs() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(utils.NextPollInterval(&pollIntervalBase))
		}
	}
}
