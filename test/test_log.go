package test

import (
	"fmt"

	"github.com/NeilXu2017/landau/config"
	"github.com/NeilXu2017/landau/log"
)

var (
	logConfigContent = `
	{
		"console": {
			"enable": true,
			"level": "CRITICAL"
		},  
		"files": [
			{ 
				"enable": true,
				"level": "DEBUG",
				"filename":"api.log",
				"category": "API",
				"pattern": "[%D %T] [%L] %M",
				"rotate": true,								
				"maxsize": "500M",
				"daily": true,
				"stdout":false
			},
			{ 
				"enable": true,
				"level": "DEBUG",
				"filename":"main.log",
				"category": "main",
				"pattern": "[%D %T] [%L] %M",
				"rotate": true,								
				"maxsize": "500M",
				"daily": true,
				"stdout":false
			},
			{ 
				"enable": true,
				"level": "DEBUG",
				"filename":"gin.log",
				"category": "gin",
				"pattern": "[%D %T] [%L] %M",
				"rotate": true,								
				"maxsize": "500M",
				"daily": true,
				"stdout":false
			},     
			{
				"enable": true,
				"level": "DEBUG",
				"filename":"server_json.log",
				"category": "jsonLogFile",
				"pattern": "%M",
				"rotate": false,								
				"maxsize": "500M",
				"daily": true
			}
		],
		"stdout2File":"jsonLogFile"
	}	
	`
	clientLogConfigContent = `
	{
		"console": {
			"enable": true,
			"level": "CRITICAL"
		},  
		"files": [
			{ 
				"enable": true,
				"level": "DEBUG",
				"filename":"client.log",
				"category": "main",
				"pattern": "[%D %T] [%L] %M",
				"rotate": true,								
				"maxsize": "500M",
				"daily": true,
				"stdout":false
			},
			{
				"enable": true,
				"level": "DEBUG",
				"filename":"client_json.log",
				"category": "jsonLogFile",
				"pattern": "%M",
				"rotate": false,								
				"maxsize": "500M",
				"daily": true
			}
		],
		"HideStdout2File":"jsonLogFile"
	}	
	`
)

// InitLog 测试Log初试化
func InitLog() {
	log.LoadLogConfig(clientLogConfigContent, "main")
	//log.LoadLogConfig(logConfigContent, "main")
}

// CheckLog 测试
func CheckLog() {
	fmt.Printf("\n\nCHECK LOG:\n")
	log.Info("HELLO TEST LOG Info")
	log.Debug("HELLO TEST LOG Debug")
	log.Error("HELLO TEST LOG Error")
	log.Warn("HELLO TEST LOG Warn")

	log.Info2("main", "HELLO TEST LOG Info main")
	log.Debug2("API", "HELLO TEST LOG Debug API")
	log.Error2("gin", "HELLO TEST LOG Error gin")
	log.Warn2("main", "HELLO TEST LOG Warn main")

	c := &testConfig{}
	_ = config.LoadYamlConfig2("./test/test.yaml", c)
	log.Info("YOUR CONFIG:%v", c)
}
