package entry

import (
	"github.com/NeilXu2017/landau/api"
	"github.com/NeilXu2017/landau/util"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type (
	//LandauServer 服务实例.
	LandauServer struct {
		ParseArgs                         func()                                          //解析命令行参数
		LogConfig                         string                                          //log配置内容或者有配置内容的文件路径名
		DefaultLoggerName                 string                                          //缺省logger名称
		GinLoggerName                     string                                          //gin的logger名称
		GinReleaseMode                    bool                                            //gin 是否是release模式
		HTTPServiceAddress                string                                          //提供HTTP服务的IP地址
		HTTPServicePort                   int                                             //提供HTTP服务的端口
		GRPCServiceAddress                string                                          //提供gRPC服务的IP地址
		GRPCServicePort                   int                                             //提供gRPC服务的端口
		CustomInit                        func()                                          //gin/grpc 服务启动前初试化
		GetCronTasks                      func() (interface{}, []util.SingletonCronTask)  //需要启动的定时任务，定时任务通过反射对象的方法来启动，方法名必须符合导出约束：即首字符必须大写. 对象实例(必须是指针)可以提供,也可以使用任务定义里的函数或对象实例.
		RegisterHTTPHandles               func()                                          //注册HTTP服务入口
		RegisterHTTPCustomHandles         func(route *gin.Engine)                         //注册HTTP自定义服务入口
		RegisterGRPCHandle                func(server *grpc.Server)                       //注册GRPC服务入口
		HTTPNeedCheckACL                  bool                                            //HTTP服务是否启用权限检查
		HTTPCheckACL                      api.HTTPCheckACL                                //HTTP服务权限检查函数
		HTTPEnableCustomLogTag            bool                                            //HTTP服务日志是否记录自定义Tag
		HTTPCustomLog                     api.HTTPCustomLogTag                            //HTTP服务日志自定义Tag生成
		grpcServer                        *grpc.Server                                    //GRPC 服务引擎，内部生成维护
		ginRouter                         *gin.Engine                                     //HTTP服务引擎，内部生成维护
		HTTPAuditLog                      api.HTTPAuditLog                                //审核日志记录
		PostBindingComplex                [2]string                                       //需要支持复杂JSON Unmarshal 的请求，第一个元素URL,第二个元素Action,多个值用逗号分割
		UnRegisterHTTPHandle              api.HTTPHandleFunc                              //未注册的Action Handle 处理入口
		DynamicHTTPServiceAddress         func() string                                   //修改HTTP服务的IP地址,使用场景:检测本机内网IP，更换原先配置的 127.0.0.1 地址形式
		EnablePrometheusMetric            bool                                            //是否启动Prometheus Metric 服务提供监测
		PrometheusMetricHost              string                                          //Prometheus Metric 服务IP地址, 默认与 HTTPServiceAddress一致
		PrometheusMetricPort              int                                             //Prometheus Metric 服务端口,默认 HTTPServicePort+3000
		PrometheusMetricNamespace         string                                          //Prometheus Metric 上报指标 namespace, 默认值为空
		DisableGracefulStopping           bool                                            //禁止优雅停止服务,默认值为 false, 启用优雅stopping
		GracefulTimeout                   uint64                                          //优雅stopping 等待超时时间, 默认值: 60秒
		DynamicReloadConfig               func()                                          //通过信号机制触发回调用户函数,一般用于重新加载配置, window平台不支持信号. 使用了 SIGUSR1 信号,需要同时 DisableGracefulStopping=false 时生效
		DisableServiceHealthReceiver      bool                                            //是否禁用 service health receiver 接口
		CheckServiceHealth                func() map[string][]string                      //需要进行健康检查的 service
		CheckServiceHealth2               func() (map[string][]string, map[string]string) //需要进行健康检查的 service,service 有第2个地址
		CheckServiceHealthPeriod          int                                             //CheckServiceHealth 检查周期
		DisableCheckServiceHealthSourceIp bool                                            //disableAssignSourceIp option
		ServiceName                       string                                          //自身服务名称
		ServiceAddress                    string                                          //自身服务地址
		DisableTraceServiceAddress        bool                                            //是否禁用 从request记录 service/service address
		SecondaryServiceAddress           string                                          //secondary ip service address
		EnableMonitorAPI                  bool                                            //是否开启 API 非预期返回的结果监控上报
		NotifyAPIWeChatRobot              string                                          //上报 Robot 地址 当 EnableMonitorAPI &&  EnableMonitorAPI 非空时,检测 response 是否实现了上报接口
	}
)

const (
	//DefaultLogger 缺省logger
	DefaultLogger = "main"
	//DefaultGinLogger 缺省gin使用的logger
	DefaultGinLogger = "gin"
)
