package data

import (
	"context"
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/util"
	"google.golang.org/grpc"
)

type (
	//NewGRPCClient 构建解析具体gRPC协议的客户端，此类有协议工具生成
	NewGRPCClient func(conn *grpc.ClientConn) interface{}
	GRPCService   struct {
		address               string
		newGRPCClient         NewGRPCClient
		maxReceiveMessageSize int
		logger                string
		logResponse           func(response interface{}) string
	}
)

var (
	defaultGRPCMaxReceiveMessageSize = 32 * 1024 * 1024
	defaultGRPCLogger                = "main"
	defaultGRPCTimeout               = 15
	defaultGRPCResponseShowDetail    = false
	defaultGRPCResponseShowSize      = 512
	defaultGRPCLogResponse           = func(response interface{}) string {
		msg := fmt.Sprintf("%v", response)
		if defaultGRPCResponseShowDetail {
			return msg
		}
		if utf8.RuneCountInString(msg) <= defaultGRPCResponseShowSize {
			return msg
		}
		responseRune := []rune(msg)
		return fmt.Sprintf("%s......%s", string(responseRune[0:64]), string(responseRune[len(responseRune)-64:]))
	}
)

// NewGRPCCaller 构建对象
func NewGRPCCaller(address string, newGRPCClient NewGRPCClient, maxReceiveMessageSize int, logger string, logResponse func(response interface{}) string) *GRPCService {
	g := &GRPCService{
		address:               address,
		newGRPCClient:         newGRPCClient,
		maxReceiveMessageSize: maxReceiveMessageSize,
		logger:                logger,
		logResponse:           logResponse,
	}
	return g
}

// NewGRPCCaller2 构建对象
func NewGRPCCaller2(address string, newGRPCClient NewGRPCClient) *GRPCService {
	return NewGRPCCaller(address, newGRPCClient, defaultGRPCMaxReceiveMessageSize, defaultGRPCLogger, defaultGRPCLogResponse)
}

// CallGRPCService 请求gRPC 服务接口,requestParam 必须是指针类型
func (c *GRPCService) CallGRPCService(serviceName string, requestParam interface{}, timeout int) (interface{}, error) {
	start := time.Now()
	defer func() {
		if err := recover(); err != nil {
			log.Error2(c.logger, "[GRPC]\t[%s]\t[%s]\tRequest:%v\tPanic:%v", time.Since(start), serviceName, requestParam, err)
		}
	}()
	grpcAddress := util.AddrConvert(c.address, util.IPV6Bracket)
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error2(c.logger, "[GRPC]\t[%s]\t[%s]\tRequest:%v\tDial Error:%v", time.Since(start), serviceName, requestParam, err)
		return nil, err
	}
	defer conn.Close()
	client := c.newGRPCClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	v := reflect.ValueOf(client)
	m := v.MethodByName(serviceName)
	params := make([]reflect.Value, 3)
	params[0] = reflect.ValueOf(ctx)
	params[1] = reflect.ValueOf(requestParam)
	params[2] = reflect.ValueOf(grpc.MaxCallRecvMsgSize(c.maxReceiveMessageSize))
	rs := m.Call(params)
	var callError error
	if len(rs) > 1 {
		if r2 := rs[1].Interface(); r2 != nil {
			callError = r2.(error)
		}
	}
	r1 := rs[0].Interface()
	if callError == nil {
		log.Info2(c.logger, "[GRPC]\t[%s]\t[%s]\tRequest:%v\tResponse:%s", time.Since(start), serviceName, requestParam, c.logResponse(r1))
	} else {
		strResponse := ""
		v1 := reflect.ValueOf(r1)
		if !v1.IsNil() {
			strResponse = c.logResponse(r1)
		}
		log.Error2(c.logger, `[GRPC]	[%s]	[%s]	Request:%v	Response:%s	Error:%v`, time.Since(start), serviceName, requestParam, strResponse, callError)
	}
	return r1, callError
}

// CallGRPCService2 请求gRPC 服务接口,requestParam 必须是指针类型
func (c *GRPCService) CallGRPCService2(serviceName string, requestParam interface{}) (interface{}, error) {
	return c.CallGRPCService(serviceName, requestParam, defaultGRPCTimeout)
}
