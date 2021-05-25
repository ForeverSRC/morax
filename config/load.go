package config

import (
	"context"
	"fmt"
	"log"
	"os"
)

import (
	ck "github.com/ForeverSRC/morax/config/check"
	cc "github.com/ForeverSRC/morax/config/consumer"
	cl "github.com/ForeverSRC/morax/config/logger"
	cp "github.com/ForeverSRC/morax/config/provider"
	cr "github.com/ForeverSRC/morax/config/registry"
	cs "github.com/ForeverSRC/morax/config/service"
	"github.com/ForeverSRC/morax/logger"
	"github.com/ForeverSRC/morax/registry/consul"
	"github.com/ForeverSRC/morax/service"
)

import (
	"github.com/spf13/viper"
)

var v = viper.New()

func Load(ctx context.Context) *service.MoraxService {
	loadConf()
	initLogger()

	ms := new(service.MoraxService)

	initService(ms, ctx)
	initHealthCheck(ms)

	initRegistryClient()

	initConsumer(ms)
	initProvider(ms)
	return ms
}

func loadConf() {
	// read file from path which specified in environment variable "CONF_FILE_PATH"
	v.SetConfigType("yaml")
	path := os.Getenv("CONF_FILE_PATH")
	v.SetConfigFile(path)
	err := v.ReadInConfig()
	if err != nil {
		log.Fatal("read config file error:", err)
	}
}

func genConfigInfo(key string, confPtr interface{}, ignoreAble bool) (bool, error) {
	part := v.Sub(key)
	if part == nil {
		if ignoreAble {
			return false, nil
		} else {
			return false, fmt.Errorf("config file error: no %s info", key)
		}
	}
	err := part.Unmarshal(confPtr)
	if err != nil {
		return false, fmt.Errorf("generate %s config info error: %s", key, err)
	}
	return true, nil
}

func initLogger() {
	lgcf := &cl.LoggerConfig{}
	res, err := genConfigInfo("logger", lgcf, false)
	if err != nil {
		log.Fatal(err)
	}
	if !res {
		return
	}

	logger.NewLogger(lgcf)

}

func initService(ms *service.MoraxService, ctx context.Context) {
	sf := &cs.ServiceConfig{}
	res, err := genConfigInfo("service", sf, false)
	if err != nil {
		log.Fatal(err)
	}
	if !res {
		return
	}

	err = ms.InitService(sf, ctx)
	if err != nil {
		log.Fatal("init service error: ", err)
	}
}

func initHealthCheck(ms *service.MoraxService) {
	ckf := &ck.CheckConfig{}
	res, err := genConfigInfo("check", ckf, false)
	if err != nil {
		log.Fatal(err)
	}
	if !res {
		return
	}

	ms.InitHealthCheck(ckf)
}

func initRegistryClient() {
	dcf := &cr.ConsulClientConfig{}
	res, err := genConfigInfo("registries", dcf, false)
	if err != nil {
		log.Fatal(err)
	}
	if !res {
		return
	}

	consul.NewClient(dcf)
}

func initConsumer(ms *service.MoraxService) {
	cmf := &cc.ConsumerConfig{}
	res, err := genConfigInfo("consumer", cmf, true)
	if err != nil {
		log.Fatal(err)
	}
	if !res {
		return
	}

	ms.InitRpcConsumer(cmf)

}

func initProvider(ms *service.MoraxService) {
	pvf := &cp.ProviderConfig{}
	res, err := genConfigInfo("provider", pvf, true)
	if err != nil {
		log.Fatal(err)
	}
	if !res {
		return
	}

	ms.InitRpcProvider(pvf)
}
