package consumer

import (
	"github.com/morax/common/constants"
	"github.com/morax/common/utils"
	cc "github.com/morax/config/consumer"
	"strings"
)

type MethodInfo struct {
	ProviderName string
	MethodName   string
	cc.ConfInfo
}

func (mi *MethodInfo) SetConfigInfo(c *cc.ConsumerConfig) {
	mi.LBType = c.Reference.LBType
	mi.Timeout = c.Reference.Timeout
	mi.Retries = c.Reference.Retries

	vp, ok := c.Reference.Providers[mi.ProviderName]
	if ok {
		mi.LBType = utils.If(vp.LBType != "", vp.LBType, mi.LBType).(string)
		mi.Timeout = utils.If(vp.Timeout != 0, vp.Timeout, mi.Timeout).(int)
		mi.Retries = utils.If(vp.Retries != 0, vp.Retries, mi.Retries).(int)

		vm, ok := vp.Methods[strings.ToLower(mi.MethodName)]
		if ok {
			mi.LBType = utils.If(vm.LBType != "", vm.LBType, mi.LBType).(string)
			mi.Timeout = utils.If(vm.Timeout != 0, vm.Timeout, mi.Timeout).(int)
			mi.Retries = utils.If(vm.Retries != 0, vm.Retries, mi.Retries).(int)
		}
	}

	mi.LBType = utils.If(mi.LBType == "", constants.DefaultLoadBalance, mi.LBType).(string)
	mi.Timeout = utils.If(mi.Timeout == 0, constants.DefaultTimeOut, mi.Timeout).(int)
}
