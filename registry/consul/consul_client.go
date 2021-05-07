package consul

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type HelloService struct {
	SayHello func(req interface{}) interface{}

}

type Invoker struct {
	method func(p ...interface{}) interface{}
	request struct{}
	response struct{}
}

type DiscoveryClient struct {
	host string
	port int
	url string
}

func NewClient(config DiscoveryClientConfig)* DiscoveryClient {
	client:=&DiscoveryClient{host: config.Host,port: config.Port}
	client.url="http://"+client.host+":"+strconv.Itoa(client.port)+"/v1/agent/service/register"
	return client
}

func (dc *DiscoveryClient) Register(info *InstanceConfig) error{

	byteData,err:=json.Marshal(info)

	if err != nil {
		log.Printf("json format err: %s", err)
		return err
	}

	req, err := http.NewRequest("PUT", dc.url,bytes.NewReader(byteData))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{}
	client.Timeout = time.Second * 2
	resp, err := client.Do(req)

	if err != nil {
		log.Printf("register service err : %s", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("register service http request errCode : %v", resp.StatusCode)
		return fmt.Errorf("register service http request errCode : %v", resp.StatusCode)
	}

	log.Println("register service success")
	return nil

}


