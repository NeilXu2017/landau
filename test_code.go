package main

import (
	"flag"
	"fmt"
	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/test"
	"github.com/NeilXu2017/landau/version"
)

var (
	testAction = flag.String("test_action", "", "test action: engine,cron_job,normal,normal_server,unit")
	unitAction = flag.String("unit_action", "", "unit action:")
)

func main() {
	flag.Parse()
	version.ShowVersion()
	switch *testAction {
	case "engine":
		test.CheckEngine()
	case "cron_jon":
		test.CheckCronJobMode()
	case "normal":
		test.CheckNormalMode()
	case "normal_server":
		test.CheckNormalServerMode()
	case "unit":
		unitTest()
	case "learn":
		learnTestCode()
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
