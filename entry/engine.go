package entry

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/NeilXu2017/landau/api"
	"github.com/NeilXu2017/landau/data"
	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/prometheus"
	"github.com/NeilXu2017/landau/util"
	"github.com/NeilXu2017/landau/version"
	sysLog "log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	srv            *http.Server                                        //saved to be used in graceful stopping
	secondSrv      *http.Server                                        //saved to be used in graceful stopping
	grpcServer     *grpc.Server                                        //saved to be used in graceful stopping
	reload         = flag.Bool("reload", false, "Signal reload event") //reload cmd
	reloadCallback func()                                              //reload 回调
)

// Start 服务启动入口,作为 web server 或 grpc server
func (c *LandauServer) Start() {
	flag.Parse()
	if c.ParseArgs != nil {
		c.ParseArgs()
	}
	version.ShowVersion()
	if !c.DisableGracefulStopping && c.DynamicReloadConfig != nil {
		reloadCallback = c.DynamicReloadConfig
	}
	makeReloadSignal()
	log.LoadLogConfig(c.LogConfig, c.DefaultLoggerName)
	if c.GinLoggerName != "" {
		gin.DefaultWriter = log.NewConsoleLogger(c.GinLoggerName)
		gin.DefaultErrorWriter = log.NewConsoleLogger(c.GinLoggerName)
	}
	if c.GinReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}
	if c.CustomInit != nil {
		c.CustomInit()
	}
	if c.HTTPServicePort > 0 || c.GRPCServicePort > 0 {
		if c.GetCronTasks != nil {
			p, jobs := c.GetCronTasks()
			util.StartCronJob(p, jobs)
		}
		if c.GRPCServicePort > 0 {
			c.grpcServer = grpc.NewServer()
			grpcServer = c.grpcServer
			c.RegisterGRPCHandle(c.grpcServer)
			reflection.Register(c.grpcServer)
			startGRPC := func() {
				address := fmt.Sprintf("%s:%d", util.IPConvert(c.GRPCServiceAddress, util.IPV6Bracket), c.GRPCServicePort)
				log.Info("[gRPC] Listen address:%s", address)
				if gRPCListen, err := net.Listen("tcp", address); err == nil {
					if serverErr := c.grpcServer.Serve(gRPCListen); serverErr != nil {
						sysLog.Fatalf("[gRPC] Start gRPC server error,err:%v", serverErr)
					}
				} else {
					sysLog.Fatalf("[gRPC] listen gRPC address error,err:%v", err)
				}
			}
			if c.HTTPServicePort > 0 {
				go startGRPC()
			} else {
				if c.DisableGracefulStopping {
					startGRPC()
				} else {
					go startGRPC()
					gracefulStop(c.GracefulTimeout)
				}
			}
		}
		if c.HTTPServicePort > 0 {
			c.ginRouter = gin.Default()
			if c.RegisterHTTPHandles != nil {
				c.RegisterHTTPHandles()
			}
			if c.RegisterHTTPCustomHandles != nil {
				c.RegisterHTTPCustomHandles(c.ginRouter)
			}
			if !c.DisableServiceHealthReceiver {
				healthReceiverLog := func(response interface{}) string { return fmt.Sprintf("%v", response) }
				api.AddHTTPHandle("/ServiceHealthCheck", "ServiceHealthCheck", data.NewServiceHealthCheckRequest, data.DoHealthCheck, healthReceiverLog, "health_receiver")
				c.ginRouter.GET("/output_keepalived_trace", data.OutputKeepaliveStatics)
			}
			api.DisableTraceServiceAddress = c.DisableTraceServiceAddress
			data.ServiceName = c.ServiceName
			api.SetPostBindingComplex(c.PostBindingComplex)
			api.SetUnRegisterHandle(c.UnRegisterHTTPHandle)
			api.RegisterHTTPHandle(c.ginRouter)
			api.RegisterRestfulHTTPHandle(c.ginRouter)
			api.SetHTTPCheckACL(c.HTTPNeedCheckACL, c.HTTPCheckACL)
			api.SetHTTPCustomLogTag(c.HTTPEnableCustomLogTag, c.HTTPCustomLog)
			api.SetHTTPAuditLog(c.HTTPAuditLog)
			addr := c.HTTPServiceAddress
			if c.DynamicHTTPServiceAddress != nil {
				addr = c.DynamicHTTPServiceAddress()
			}
			prometheus.SetNamespace(c.PrometheusMetricNamespace)
			if c.PrometheusMetricHost != "" {
				prometheus.SetServerHost(c.PrometheusMetricHost)
			} else {
				prometheus.SetServerHost(addr)
			}
			if c.PrometheusMetricPort > 0 {
				prometheus.SetServerPort(c.PrometheusMetricPort)
			} else {
				prometheus.SetServerPort(c.HTTPServicePort + 3000)
			}
			if c.EnablePrometheusMetric {
				go prometheus.StartApiMetric()
			}
			if addr != "" && addr != "0.0.0.0" && addr != "::" {
				data.LocalPrimaryAddress = addr
			}
			address := fmt.Sprintf("%s:%d", util.IPConvert(addr, util.IPV6Bracket), c.HTTPServicePort)
			data.ServiceAddress = address
			secondaryAddress := ""
			if c.SecondaryServiceAddress != "" {
				secondaryAddress = fmt.Sprintf("%s:%d", util.IPConvert(c.SecondaryServiceAddress, util.IPV6Bracket), c.HTTPServicePort)
				data.SecondaryServiceAddress = secondaryAddress
				if c.SecondaryServiceAddress != "" && c.SecondaryServiceAddress != "0.0.0.0" && c.SecondaryServiceAddress != "::" {
					data.LocalSecondaryAddress = c.SecondaryServiceAddress
				}
			}
			if c.CheckServiceHealth != nil || c.CheckServiceHealth2 != nil {
				if c.CheckServiceHealthPeriod > 0 {
					data.MonitorServiceAddrPeriod = c.CheckServiceHealthPeriod
				}
				data.MonitorServiceAddrChange2 = c.CheckServiceHealth2
				data.MonitorServiceAddrChange = c.CheckServiceHealth
				data.RegisterServiceHealth()
				go data.MonitorServiceHealthConfigs()
				go data.StartHealthChecking()
			}
			log.Info("[HTTP] Listen address:%s", address)
			if secondaryAddress != "" {
				log.Info("[HTTP] Listen secondary address:%s", secondaryAddress)
			}
			if c.DisableGracefulStopping {
				if secondaryAddress != "" {
					secondSrv = &http.Server{Addr: secondaryAddress, Handler: c.ginRouter}
					go func() {
						if err := secondSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
							sysLog.Fatalf("[HTTP] Start gin server error,err:%v", err)
						}
					}()
				}
				if err := c.ginRouter.Run(address); err != nil {
					sysLog.Fatalf("[HTTP] Start gin server error,err:%v", err)
				}
			} else {
				srv = &http.Server{Addr: address, Handler: c.ginRouter}
				go func() {
					if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
						sysLog.Fatalf("[HTTP] Start gin server error,err:%v", err)
					}
				}()
				if secondaryAddress != "" {
					secondSrv = &http.Server{Addr: secondaryAddress, Handler: c.ginRouter}
					go func() {
						if err := secondSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
							sysLog.Fatalf("[HTTP] Start gin server error,err:%v", err)
						}
					}()
				}
				gracefulStop(c.GracefulTimeout)
			}
		}
	}
	log.Close()
}

// StartCronJobMode 作为普通程序启动(仅仅运行cron job)
func (c *LandauServer) StartCronJobMode(gracefulTimeout uint64) {
	flag.Parse()
	if c.ParseArgs != nil {
		c.ParseArgs()
	}
	version.ShowVersion()
	if !c.DisableGracefulStopping && c.DynamicReloadConfig != nil {
		reloadCallback = c.DynamicReloadConfig
	}
	makeReloadSignal()
	log.LoadLogConfig(c.LogConfig, c.DefaultLoggerName)
	if c.CustomInit != nil {
		c.CustomInit()
	}
	if c.GetCronTasks != nil {
		p, jobs := c.GetCronTasks()
		util.StartCronJob(p, jobs)
		if util.ScheduledJobCount > 0 {
			log.Info("[CronJobMode] running...")
			if gracefulTimeout == 0 {
				gracefulTimeout = 60
			}
			gracefulStop(gracefulTimeout)
		}
	}
	log.Close()
}

// StartNormalServerMode 作为普通服务程序启动,运行 mainEntry
func (c *LandauServer) StartNormalServerMode(mainEntry func(), gracefulTimeout uint64) {
	flag.Parse()
	if c.ParseArgs != nil {
		c.ParseArgs()
	}
	version.ShowVersion()
	if !c.DisableGracefulStopping && c.DynamicReloadConfig != nil {
		reloadCallback = c.DynamicReloadConfig
	}
	makeReloadSignal()
	log.LoadLogConfig(c.LogConfig, c.DefaultLoggerName)
	if c.CustomInit != nil {
		c.CustomInit()
	}
	if c.GetCronTasks != nil {
		p, jobs := c.GetCronTasks()
		util.StartCronJob(p, jobs)
	}
	log.Info("[NormalServerMode] %v running...", mainEntry)
	go mainEntry() //mainEntry 退出, 程序也不退出,等待信号退出
	if gracefulTimeout == 0 {
		gracefulTimeout = 60
	}
	gracefulStop(gracefulTimeout)
	log.Close()
}

// StartNormalMode 作为普通服务程序启动
func (c *LandauServer) StartNormalMode(mainEntry func()) {
	flag.Parse()
	if c.ParseArgs != nil {
		c.ParseArgs()
	}
	version.ShowVersion()
	log.LoadLogConfig(c.LogConfig, c.DefaultLoggerName)
	if c.CustomInit != nil {
		c.CustomInit()
	}
	log.Info("[NormalMode] %v running...", mainEntry)
	mainEntry()
	log.Close()
}

func gracefulStop(gracefulTimeout uint64) {
	waitMaxSecond := gracefulTimeout
	if waitMaxSecond == 0 {
		waitMaxSecond = 60
	}
	waitingShutdownServer := func() {
		wg := sync.WaitGroup{}
		httpSrvShutdown := func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(waitMaxSecond))
			defer cancel()
			if srv != nil {
				if err := srv.Shutdown(ctx); err != nil {
					sysLog.Fatalf("[HTTP] Server Shutdown error,err:%v", err)
				}
			}
		}
		secondHttpSrvShutdown := func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(waitMaxSecond))
			defer cancel()
			if secondSrv != nil {
				if err := secondSrv.Shutdown(ctx); err != nil {
					sysLog.Fatalf("[HTTP] Server Shutdown error,err:%v", err)
				}
			}
		}
		cronJobShutdown := func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(waitMaxSecond))
			defer cancel()
			if err := util.CronJobShutdown(ctx); err != nil {
				sysLog.Fatalf("[CronJobManager] Shutdown error,err:%v", err)
			}
		}
		grpcSvrShutdown := func() {
			defer wg.Done()
			if grpcServer != nil {
				grpcServer.GracefulStop()
			}
		}
		wg.Add(4)
		go httpSrvShutdown()
		go secondHttpSrvShutdown()
		go cronJobShutdown()
		go grpcSvrShutdown()
		data.NotifyCheckerShutdown()
		wg.Wait()
	}
	monitorSignal := make(chan os.Signal)
	if reloadCallback != nil {
		signal.Notify(monitorSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1)
	} else {
		signal.Notify(monitorSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	}
	for i := range monitorSignal {
		switch i {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			log.Info("[Engine] receive exit signal %s, Shutdown Server ...", i.String())
			waitingShutdownServer()
			log.Info("[Engine] Shutdown Server completed.")
			return
		case syscall.SIGUSR1:
			log.Info("[Engine] receive usr1 signal, dispatch reload event now")
			if reloadCallback != nil {
				reloadCallback()
			}
		}
	}
}

func makeReloadSignal() {
	if *reload {
		if pid := os.Getegid(); pid != -1 {
			appName := os.Args[0]
			if runtime.GOOS == "windows" {
				appName = strings.Replace(appName, "\\", "/", -1)
			}
			if i := strings.LastIndex(appName, "/"); i > 0 {
				appName = appName[i+1:]
			}
			appFullPath := os.Args[0]
			cmd := exec.Command("ps", "-e")
			if out, err := cmd.CombinedOutput(); err == nil {
				processes := strings.Split(string(out), "\n")
				for _, process := range processes {
					if id, n, err := getProcIdName(process); err == nil && id != pid && (n == appName || n == appFullPath) {
						_ = syscall.Kill(id, syscall.SIGUSR1)
						break
					}
				}
			}
		}
		os.Exit(0)
	}
}

func getProcIdName(process string) (int, string, error) {
	p := strings.TrimSpace(process)
	lines := strings.Split(p, " ")
	if len(lines) >= 4 {
		pid, err := strconv.Atoi(lines[0])
		if err != nil {
			return -1, "", err
		}
		return pid, lines[len(lines)-1], nil
	}
	return -1, "", fmt.Errorf("pare proceess (%s) invalid", p)
}
