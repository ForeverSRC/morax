package consumer

import (
	"context"
	"errors"
	"fmt"
	"github.com/ForeverSRC/morax/registry/consul"
	"reflect"
	"sync"
	"time"
)

import (
	"github.com/ForeverSRC/morax/common/types"
	"github.com/ForeverSRC/morax/common/utils"
	cc "github.com/ForeverSRC/morax/config/consumer"
	. "github.com/ForeverSRC/morax/error"
	"github.com/ForeverSRC/morax/logger"
)

type RpcConsumer struct {
	conf *cc.ConsumerConfig
	// providers 订阅的服务提供者集合 providerName->instances
	providers      map[string]*ProviderInstances
	inShutdown     types.AtomicBool
	mu             sync.Mutex
	ctx            context.Context
	watcherCtx     map[*context.Context]context.CancelFunc
	allClientClose bool
}

var consumer *RpcConsumer

func NewRpcConsumer(config *cc.ConsumerConfig) {
	consumer = consumer.NewRpcConsumer(context.Background(), config)
}

func (c *RpcConsumer) NewRpcConsumer(ctx context.Context, config *cc.ConsumerConfig) *RpcConsumer {
	con := &RpcConsumer{
		conf:       config,
		providers:  make(map[string]*ProviderInstances),
		ctx:        ctx,
		watcherCtx: make(map[*context.Context]context.CancelFunc),
	}
	con.inShutdown.SetFalse()
	return con
}

func Shutdown(ctx context.Context) error {
	return consumer.Shutdown(ctx)
}

func (c *RpcConsumer) Shutdown(ctx context.Context) error {
	// 设置标志位
	c.inShutdown.SetTrue()
	c.mu.Lock()
	// 关闭所有rpc client
	// net/rpc包中 Client的close方法会通过加锁的机制，阻塞等待当前send完成
	c.closeAllClientLock()
	// 停止所有watcher
	for _, cancel := range c.watcherCtx {
		cancel()
	}
	// 关闭consul client的idle connection
	consul.CloseIdleConn()
	c.mu.Unlock()

	pollIntervalBase := time.Millisecond
	timer := time.NewTimer(utils.NextPollInterval(&pollIntervalBase))
	defer timer.Stop()
	for {
		// 没有打开的rpc client
		if c.allClientClose {
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

func (c *RpcConsumer) closeAllClientLock() {
	for _, p := range c.providers {
		for _, client := range p.instances {
			_ = client.Close()
		}
	}
	c.allClientClose = true
}

func RegistryConsumer(name string, service interface{}) error {
	return consumer.RegistryConsumer(name, service)
}

func (c *RpcConsumer) RegistryConsumer(name string, service interface{}) error {
	//获得传入结构体指针实际指向的结构体
	s := reflect.ValueOf(service).Elem()
	if s.Kind() != reflect.Struct {
		return errors.New("invalid consumer service type")
	}

	for i := 0; i < s.NumField(); i++ {
		// 函数
		field := s.Field(i)
		rTyp, er := checkMethodField(&field)
		if er != nil {
			logger.Error("check method field error: %s", er)
			continue
		}

		// methodName属于结构体字段名
		methodName := s.Type().Field(i).Name
		serviceMethod := fmt.Sprintf("%s.%s", name, methodName)

		info := MethodInfo{
			ProviderName: name,
			MethodName:   methodName,
		}
		// 获取当前方法的配置：负载均衡策略，超时重试
		info.SetConfigInfo(c.conf)

		mf := reflect.MakeFunc(field.Type(), func(args []reflect.Value) []reflect.Value {
			// consumer处于shutdown阶段时停止一切调用，返回错误
			if c.inShutdown.IsSet() {
				return []reflect.Value{reflect.Zero(*rTyp), reflect.ValueOf(RpcError{
					Err: fmt.Errorf("consumer is shutting down"),
				})}
			}

			callSuccessCh := make(chan []reflect.Value)
			callFailCh := make(chan error)
			defer close(callFailCh)
			defer close(callSuccessCh)

			core := func(ctx context.Context) <-chan []reflect.Value {
				defer func() {
					if e := recover(); e != nil {
						logger.Error("recover: rpc call panic:%v", e)
					}
				}()

				parseErr := func(err error) {
					callFailCh <- err
				}

				resCh := make(chan []reflect.Value, 1)
				// 服务发现
				providerInstances, ok := c.providers[name]
				if !ok {
					parseErr(errors.New("no instance of provider: " + name))
				}

				// 负载均衡
				client, err := providerInstances.LoadBalance(info.LBType)
				if err != nil {
					parseErr(err)
					return nil
				}

				// 调用
				resp := reflect.New(*rTyp) //a pointer
				err = client.Call(serviceMethod, args[0].Interface(), resp.Interface())
				if err != nil {
					parseErr(err)
					return nil
				}

				resCh <- []reflect.Value{resp.Elem(), reflect.Zero(reflect.TypeOf(RpcError{}))}
				return resCh
			}

			invoke := func(ctx context.Context, timer *time.Timer, isRetry bool) {
				if isRetry {
					timer.Reset(time.Millisecond * time.Duration(info.Timeout))
				}

				select {
				case res := <-core(ctx):
					callSuccessCh <- res
					return
				case <-ctx.Done():
					return
				}

			}

			// 实际调用过程
			callCtx, callCancel := context.WithCancel(c.ctx)
			count := 0
			timer := time.NewTimer(time.Millisecond * time.Duration(info.Timeout))
			go invoke(callCtx, timer, false)

			for {
				select {
				case res := <-callSuccessCh:
					{
						timer.Stop()
						callCancel()
						return res

					}
				case fail := <-callFailCh:
					{
						timer.Stop()
						callCancel()
						// todo: 失败重试
						if count < info.Retries {
							count++
							callCtx, callCancel = context.WithCancel(c.ctx)
							go invoke(callCtx, timer, true)
						} else {
							return []reflect.Value{reflect.Zero(*rTyp), reflect.ValueOf(RpcError{
								Err: fail,
							})}
						}
					}
				case <-timer.C:
					{
						timer.Stop()
						callCancel()
						// todo: 超时重试
						if count < info.Retries {
							count++
							callCtx, callCancel = context.WithCancel(c.ctx)
							go invoke(callCtx, timer, true)
						} else {
							return []reflect.Value{reflect.Zero(*rTyp), reflect.ValueOf(RpcError{Err: errors.New("rpc call time out")})}
						}
					}
				}
			}
		})

		field.Set(mf)
	}

	// 设置并启动监，避免对同一个provider多次设定监听
	if _, ok := c.providers[name]; !ok {
		pss := NewProviderInstances(name)
		c.providers[name] = pss
		ctx, cancel := context.WithCancel(c.ctx)
		c.watcherCtx[&ctx] = cancel

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-pss.Watch(ctx):
					continue
				}
			}
		}(ctx)
	}

	return nil
}

func checkMethodField(field *reflect.Value) (*reflect.Type, error) {
	if field.Kind() != reflect.Func {
		return nil, errors.New("not a func field")
	}

	ft := field.Type()

	if ft.NumIn() != 1 {
		return nil, errors.New("number of input params must be only one")
	}

	if ft.NumOut() != 2 {
		return nil, errors.New("number of output params must be only two")
	}

	iTyp := ft.In(0)
	if iTyp.Kind() != reflect.Struct {
		return nil, errors.New("input params type should be a struct")
	}

	rTyp := ft.Out(0)
	if rTyp.Kind() != reflect.Struct {
		return nil, errors.New("output params type should be a struct")
	}

	rErr := ft.Out(1)
	if rErr.Kind() != reflect.Struct {
		return nil, errors.New("invalid error param")
	}

	return &rTyp, nil
}
