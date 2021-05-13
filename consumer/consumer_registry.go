package consumer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"reflect"
	"time"
)

import (
	cc "github.com/ForeverSRC/morax/config/consumer"
	. "github.com/ForeverSRC/morax/error"
	"github.com/ForeverSRC/morax/loadbalance"
	"github.com/ForeverSRC/morax/logger"
)

type RpcConsumer struct {
	conf              *cc.ConsumerConfig
	providerInstances map[string]*ConsumeServersStore
}

var consumer *RpcConsumer

func NewRpcConsumer(config *cc.ConsumerConfig) {
	consumer = &RpcConsumer{
		conf:              config,
		providerInstances: make(map[string]*ConsumeServersStore),
	}
}

func RegistryConsumer(name string, service interface{}) error {
	//获得传入结构体指针实际指向的结构体
	s := reflect.ValueOf(service).Elem()
	if s.Kind() != reflect.Struct {
		return errors.New("invalid service type")
	}
	for i := 0; i < s.NumField(); i++ {
		// 函数
		field := s.Field(i)
		rTyp, err := checkMethodField(&field)
		if err != nil {
			logger.Error("check method field error: %s", err)
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
		info.SetConfigInfo(consumer.conf)

		mf := reflect.MakeFunc(field.Type(), func(args []reflect.Value) []reflect.Value {
			callSuccessCh := make(chan []reflect.Value)
			callFailCh := make(chan error)
			ctx, cancel := context.WithCancel(context.Background())

			core := func(ctx context.Context) {
				defer func() {
					if e := recover(); e != nil {
						logger.Error("recover: rpc call panic:%v", e)
					}
				}()

				parseErr := func(err error) {
					callFailCh <- err
				}

				// 服务发现
				comsumeServs, ok := consumer.providerInstances[name]
				if !ok {
					parseErr(errors.New("no instance of provider: " + name))
				}

				instances := comsumeServs.Get()
				if err != nil {
					parseErr(err)
					return
				}

				// 负载均衡
				inst, err := loadbalance.DoBalance(info.LBType, instances)
				if err != nil {
					parseErr(err)
				}

				// 调用
				conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", inst.Address, inst.Port))
				if err != nil {
					parseErr(err)
					return
				}

				client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn))
				resp := reflect.New(*rTyp) //a pointer
				err = client.Call(serviceMethod, args[0].Interface(), resp.Interface())
				if err != nil {
					parseErr(err)
					return
				}

				callSuccessCh <- []reflect.Value{resp.Elem(), reflect.Zero(reflect.TypeOf(RpcError{}))}
			}

			invoke := func(ctx context.Context, timer *time.Timer, isRetry bool) {
				if isRetry {
					timer.Reset(time.Millisecond * time.Duration(info.Timeout))
				}

				for {
					select {
					case <-ctx.Done():
						return
					default:
						core(ctx)
						return
					}
				}
			}

			count := 0
			timer := time.NewTimer(time.Millisecond * time.Duration(info.Timeout))
			go invoke(ctx, timer, false)

			for {
				select {
				case res := <-callSuccessCh:
					{
						timer.Stop()
						cancel()
						return res

					}
				case fail := <-callFailCh:
					{
						timer.Stop()
						cancel()
						return []reflect.Value{reflect.Zero(*rTyp), reflect.ValueOf(RpcError{
							Err: fail,
						})}
					}
				case <-timer.C:
					{
						timer.Stop()
						cancel()
						if count < info.Retries {
							count++
							go invoke(ctx, timer, true)
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
	if _, ok := consumer.providerInstances[name]; !ok {
		cs := NewConsumeServersStore(name)
		consumer.providerInstances[name] = cs
		go func() {
			for {
				if !cs.Watch() {
					time.Sleep(5 * time.Second)
				}
			}
		}()
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
