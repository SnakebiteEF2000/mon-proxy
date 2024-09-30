package main

import (
	"fmt"

	"github.com/SnakebiteEF2000/mon-proxy/internal/util/cfg"
	"github.com/SnakebiteEF2000/mon-proxy/pkg/proxy"
)

func GetSocketConfigs() []proxy.ProxyConfig {
	var configs []proxy.ProxyConfig

loop:
	for i := 1; ; i++ {
		destSocket := cfg.GetEnv(fmt.Sprintf("DESTINATION_SOCKET_%d", i), "") // REMOVE THIS
		label := cfg.GetEnv(fmt.Sprintf("REQUIRED_LABEL_%d", i), "")          // make this use cmp compare

		if destSocket == "" || label == "" {
			break loop
		}

		configs = append(configs, proxy.ProxyConfig{
			DestinationSocket: destSocket,
			RequiredLabel:     label,
		})
		log.Debug("destination socket config: ", destSocket, "label: ", label)
	}
	return configs
}

func GetSocketConfigsMOCK() []proxy.ProxyConfig {
	var configs []proxy.ProxyConfig
	configs = append(configs, proxy.ProxyConfig{
		DestinationSocket: "/tmp/filtered-docker-1.sock",
		RequiredLabel:     "label1=true",
	})
	log.Criticalf("!!! DEBUG ONLY VALUES USED !!! DEBUG Destination socket: %s, DEBUG Required label: %s !!! DO NOT USE !!!", configs[0].DestinationSocket, configs[0].RequiredLabel)
	return configs
}
