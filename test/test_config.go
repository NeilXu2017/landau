package test

import (
	"fmt"
	"strings"

	"github.com/NeilXu2017/landau/config"
)

type (
	testConfig struct {
		Port      int              `yaml:"port"`
		Debug     bool             `yaml:"debug"`
		Host      string           `yaml:"host"`
		RPCHost   string           `yaml:"rpc_host"`
		RPCPort   int              `yaml:"rpc_port"`
		RegionID  int              `yaml:"region_id" env:"landau_region_id"`
		LogConfig testConfigLog    `yaml:"log"`
		CronTasks []testConfigCron `yaml:"cron"`
	}
	testConfigLog struct {
		ConfigFile    string `yaml:"config_file"`
		DefaultLogger string `yaml:"default_logger" env:"landau_default_logger"`
		GinLogger     string `yaml:"gin_logger"`
	}
	testConfigCron struct {
		Name     string `yaml:"name"`
		Enable   bool   `yaml:"enable"`
		Schedule string `yaml:"schedule"`
		Func     string `yaml:"func"`
	}
)

// CheckConfig 测试配置Helper
func CheckConfig() {
	c := &testConfig{}
	err := config.LoadYamlConfig2("test/test.yaml", c)
	fmt.Printf("\n\nCHECK CONFIG:\n")
	fmt.Printf("%v", c)
	if err != nil {
		fmt.Printf("%v", err)
	}
}

func (c *testConfigLog) String() string {
	return fmt.Sprintf(`{"config_file":"%s","default_logger":"%s","gin_logger":"%s"}`, c.ConfigFile, c.DefaultLogger, c.GinLogger)
}

func (c *testConfigCron) String() string {
	return fmt.Sprintf(`{"name":"%s","enable":%v,"schedule":"%s","func":"%s"}`, c.Name, c.Enable, c.Schedule, c.Func)
}

func (c *testConfig) String() string {
	logString := c.LogConfig.String()
	var cronStings []string
	for _, cron := range c.CronTasks {
		cronStings = append(cronStings, cron.String())
	}
	cronString := strings.Join(cronStings, ",")
	return fmt.Sprintf(`{"port":%d,"debug":%v,"host":"%s","rpc_host":"%s","rpc_port":%d,"region_id":%d,"log":%s,"cron":[%s]}`, c.Port, c.Debug, c.Host, c.RPCHost, c.RPCPort, c.RegionID, logString, cronString)
}
