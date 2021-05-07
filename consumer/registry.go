package consumer

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"reflect"
)

import (
	. "github.com/morax/error"
)

type RpcConsumer struct {

}

func (c *RpcConsumer) RegistryConsumer(name string,service interface{}) error {
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
			log.Println("check method field error", err)
			continue
		}

		// methodName属于结构体字段名
		methodName := s.Type().Field(i).Name
		serviceMethod := fmt.Sprintf("%s.%s", name, methodName)

		mf := reflect.MakeFunc(field.Type(), func(args []reflect.Value) []reflect.Value {
			conn, err := net.Dial("tcp", "localhost:1234")
			if err != nil {
				rpcErr:=RpcError{Err: err}
				return []reflect.Value{reflect.Zero(*rTyp), reflect.ValueOf(rpcErr)}
			}
			client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn))
			resp := reflect.New(*rTyp) //a pointer

			err = client.Call(serviceMethod, args[0].Interface(), resp.Interface())
			if err != nil {
				rpcErr:=RpcError{Err: err}
				return []reflect.Value{reflect.Zero(*rTyp), reflect.ValueOf(rpcErr)}
			}

			return []reflect.Value{resp.Elem(),reflect.Zero(reflect.TypeOf(RpcError{}))}
		})
		field.Set(mf)
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

	rErr:=ft.Out(1)
	if rErr.Kind() != reflect.Struct {
		return nil, errors.New("invalid error param")
	}

	return &rTyp, nil
}
