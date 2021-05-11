package config

import (
	"log"
	"os"
)

import (
	"github.com/ForeverSRC/morax/common/utils"
	cc "github.com/ForeverSRC/morax/config/consumer"
	cl "github.com/ForeverSRC/morax/config/logger"
	cp "github.com/ForeverSRC/morax/config/provider"
	cr "github.com/ForeverSRC/morax/config/registry"
	"github.com/ForeverSRC/morax/consumer"
	"github.com/ForeverSRC/morax/logger"
	"github.com/ForeverSRC/morax/provider"
	"github.com/ForeverSRC/morax/registry/consul"
)

import (
	"github.com/spf13/viper"
)

var v = viper.New()

func Load() {
	// 读取配置到内存
	loadConf()
	// 初始化日志
	initLogger()
	// 初始化注册中心client
	initRegistryClient()
	// 初始化consumer
	initConsumer()
	// 初始化provider
	initProvider()
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

func initLogger() {
	lg := v.Sub("logger")
	if lg == nil {
		log.Fatal("config file error: no logger info")
	}

	lgcf := &cl.LoggerConfig{}
	err := lg.Unmarshal(lgcf)
	if err != nil {
		log.Fatal("init logger error: ", err)
	}

	logger.NewLogger(lgcf)

}

func initRegistryClient() {
	registries := v.Sub("registries")
	if registries == nil {
		log.Fatal("config file error: no registries info")
	}
	dcf := &cr.ConsulClientConfig{}
	err := registries.Unmarshal(dcf)
	if err != nil {
		log.Fatal("init registry error: ", err)
	}

	consul.NewClient(dcf)
}

func initConsumer() {
	cos := v.Sub("consumer")
	if cos == nil {
		return
	}

	cmf := &cc.ConsumerConfig{}
	err := cos.Unmarshal(cmf)
	if err != nil {
		log.Fatal("init consumer error: ", err)
	}

	consumer.NewRpcConsumer(cmf)

}

func initProvider() {
	prv := v.Sub("provider")
	if prv == nil {
		return
	}

	pvf := &cp.ProviderConfig{}
	err := prv.Unmarshal(pvf)
	if err != nil {
		log.Fatal("init provider error: ", err)
	}

	if pvf.Service.Host == "" {
		address, err := utils.GetLocalAddr()
		if err != nil {
			log.Fatal("init provider error: ", err)
		}

		pvf.Service.Host = address
	}

	provider.InitRpcService(pvf)

	err = consul.Register(pvf)
	if err != nil {
		log.Fatal("init provider error: ", err)
	}

}
