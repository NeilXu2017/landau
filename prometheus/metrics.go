package prometheus

import (
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type (
	DescTag struct {
		Name   string
		Help   string
		Enable bool
	}
)

var (
	_PrometheusServerHost       string                                            //Prometheus服务接口地址:IP
	_PrometheusServerPort       int                                               //Prometheus服务接口地址:端口
	_namespace                  string                                            //metric namespace, fqName prefix
	_metricUri                  = "metrics"                                       //metric handle 地址
	_pprofUri                   = "pprof"                                         //pprof handle 地址
	_VariableLabels             = []string{"ret_code", "action", "method", "uri"} //缺省variable label tag
	customPrometheusCollector   []prometheus.Collector
	_DefaultPrometheusCollector = []DescTag{
		{
			Name:   "uptime",
			Help:   "HTTP service uptime",
			Enable: true,
		},
		{
			Name:   "http_request_count_total",
			Help:   "Total number of HTTP requests made",
			Enable: true,
		},
		{
			Name:   "http_request_duration_seconds",
			Help:   "HTTP request latencies in seconds",
			Enable: true,
		},
	}
	uptime      *prometheus.CounterVec   //上线时长
	reqCount    *prometheus.CounterVec   //API请求次数
	reqDuration *prometheus.HistogramVec //API请求耗时分布
)

func SetServerHost(addr string)     { _PrometheusServerHost = addr } //从LandauServer 配置获取,无法直接调用设置
func SetServerPort(port int)        { _PrometheusServerPort = port } //从LandauServer 配置获取,无法直接调用设置
func SetNamespace(namespace string) { _namespace = namespace }       //从LandauServer 配置获取,无法直接调用设置
func SetMetricsUri(uri string)      { _metricUri = uri }             //修改默认的 uri 地址 需要在 LandauServer.Start()前调用
func SetPprofUri(uri string)        { _pprofUri = uri }              //修改默认的 uri 地址 需要在 LandauServer.Start()前调用

// SetVariableLabels 设置内置3个变量Tag名称(需要按照顺序修改):默认是 ret_code,action,method和uri 需要在 LandauServer.Start()前调用
func SetVariableLabels(labels ...string) {
	n, m := len(_VariableLabels), len(labels)
	for i := 0; i < n && i < m; i++ {
		_VariableLabels[i] = labels[i]
	}
}

// SetDefaultPrometheusCollector 设置内置3个指标配置(需要按照顺序修改)配置: 名称,描述和是否上报 需要在 LandauServer.Start()前调用
func SetDefaultPrometheusCollector(pc ...DescTag) {
	n, m := len(_DefaultPrometheusCollector), len(pc)
	for i := 0; i < n && i < m; i++ {
		_DefaultPrometheusCollector[i].Name = pc[i].Name
		_DefaultPrometheusCollector[i].Help = pc[i].Help
		_DefaultPrometheusCollector[i].Enable = pc[i].Enable
	}
}

// AddPrometheusCollector 如果需要增加自定义指标, 需要在 LandauServer.Start()前调用
func AddPrometheusCollector(cs ...prometheus.Collector) {
	customPrometheusCollector = append(customPrometheusCollector, cs...)
}

func StartApiMetric() {
	if _PrometheusServerPort <= 0 {
		fmt.Printf("[StartApiMetric] not started due to port value:%d is invalid", _PrometheusServerPort)
	}
	var pcs []prometheus.Collector
	for index, dc := range _DefaultPrometheusCollector {
		switch index {
		case 0:
			if dc.Enable {
				uptime = prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: _namespace, Name: dc.Name, Help: dc.Help}, nil)
				go func() {
					for range time.Tick(time.Second) {
						uptime.WithLabelValues().Inc() //更新上线时长
					}
				}()
				pcs = append(pcs, uptime)
			}
		case 1:
			if dc.Enable {
				reqCount = prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: _namespace, Name: dc.Name, Help: dc.Help}, _VariableLabels)
				pcs = append(pcs, reqCount)
			}
		case 2:
			if dc.Enable {
				reqDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: _namespace, Name: dc.Name, Help: dc.Help}, _VariableLabels)
				pcs = append(pcs, reqDuration)
			}
		}
	}
	pcs = append(pcs, customPrometheusCollector...)
	prometheus.MustRegister(pcs...)
	addr := net.JoinHostPort(_PrometheusServerHost, strconv.Itoa(_PrometheusServerPort))
	s := http.Server{Addr: addr, Handler: nil}
	http.Handle(fmt.Sprintf("/%s", _metricUri), promhttp.Handler())
	http.Handle(fmt.Sprintf("/%s", _pprofUri), pprof.Handler("pprof"))
	if err := s.ListenAndServe(); err != nil {
		panic(fmt.Sprintf("[StartApiMetric] listen failed address:[%s]. err:{%s}", addr, err.Error()))
	}
	fmt.Printf("[StartApiMetric] server is shutdown")
}

// UpdateApiMetric 框架调用,记录指标
func UpdateApiMetric(code int, action string, tStart time.Time, r *http.Request, uri string) {
	if uri == "" {
		uri = r.URL.Path
	}
	lvs := []string{strconv.Itoa(code), action, r.Method, uri}
	if reqCount != nil {
		reqCount.WithLabelValues(lvs...).Inc()
	}
	if reqDuration != nil {
		reqDuration.WithLabelValues(lvs...).Observe(time.Since(tStart).Seconds())
	}
}
