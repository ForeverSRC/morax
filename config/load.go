package config

import (

	"github.com/spf13/viper"
	"log"
	"os"
)

import (
	"github.com/ForeverSRC/morax/common/utils"
	"github.com/ForeverSRC/morax/consumer"
	cc "github.com/ForeverSRC/morax/config/consumer"
	cp "github.com/ForeverSRC/morax/config/provider"
	cr "github.com/ForeverSRC/morax/config/registry"
	"github.com/ForeverSRC/morax/provider"
	"github.com/ForeverSRC/morax/registry/consul"
)

var v = viper.New()

func Load() {
	// 读取配置到内存
	loadConf()
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
	log.Printf("config file path:%s\n", path)
	v.SetConfigFile(path)
	err := v.ReadInConfig()
	if err != nil {
		log.Fatal("read config file error:", err)
	}
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
		} else {
			log.Printf("current server ip: %s", address)
		}
		pvf.Service.Host = address
	}

	provider.InitRpcService(pvf)

	err = consul.Register(pvf)
	if err != nil {
		log.Fatal("init provider error: ", err)
	}

}
