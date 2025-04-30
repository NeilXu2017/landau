package main

import (
	"flag"
	"fmt"
	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/test"
	"github.com/NeilXu2017/landau/version"
)

var (
	testAction       = flag.String("test_action", "engine", "test action: engine,cron_job,normal,normal_server,unit")
	unitAction       = flag.String("unit_action", "", "unit action:")
	serviceName      = flag.String("service_name", "HostApi", "run as service name")
	servicePort      = flag.Int("service_port", 9010, "service port")
	keepaliveService = flag.String("keepalive_service", "HostClient,http://127.0.0.1:11010,http://127.0.0.1:11020", "keepalived service health")
	primaryIp        = flag.String("primary_ip", "10.64.95.53", "service primary ip address")
	secondaryIp      = flag.String("secondary_ip", "127.0.0.1", "service secondary ip address")
	//keepaliveService2 = flag.String("keepalive_service2", "HostClient,http://127.0.0.1:11010,http://127.0.0.1:11020^http://127.0.0.1:11010#http://10.64.95.53:11010,http://127.0.0.1:11020#http://10.64.95.53:11020", "checker service list")
	keepaliveService2 = flag.String("keepalive_service2", "HostClient,http://10.64.95.53:10010^http://10.64.95.53:10010#http://127.0.0.1:10010", "checker service list")
)

func main() {
	flag.Parse()
	version.ShowVersion()
	switch *testAction {
	case "engine":
		test.CheckEngine()
	case "cron_job":
		test.CheckCronJobMode()
	case "normal":
		test.CheckNormalMode()
	case "normal_server":
		test.CheckNormalServerMode()
	case "unit":
		unitTest()
	case "learn":
		learnTestCode()
	case "keepalived_client":
		test.KeepalivedClient(*servicePort)
	case "keepalived_client2":
		test.KeepalivedClient2(*servicePort, *primaryIp, *secondaryIp)
	case "keepalived_service":
		test.KeepalivedService(*serviceName, *servicePort, *keepaliveService)
	case "keepalived_service2":
		test.KeepalivedService2(*serviceName, *servicePort, *primaryIp, *secondaryIp, *keepaliveService2)
	default:
		fmt.Printf("test_action=%s is invalid.\n", *testAction)
	}
}

func learnTestCode() {
}

func unitTest() {
	flag.Parse()
	version.ShowVersion()
	test.InitLog()
	test.InitDB()
	switch *unitAction {
	case "json":
		test.CheckInnerJSONUnmarshal()
	case "go-routine":
		test.CheckThread()
	default:
		fmt.Printf("unit_action=%s do nothing.\n", *unitAction)
	}
	log.Close()
}
