package util

import (
	"fmt"
	"strings"
)

type (
	//IPBracketMode IPV6的格式化模式
	IPBracketMode int
)

const (
	//IPV6BracketNone 无方括号
	IPV6BracketNone IPBracketMode = iota
	//IPV6Bracket 有方括号
	IPV6Bracket
)

// IPConvert IP格式化输出:IPV4无变化,IPV6根据mode进行修正（有无方括号）
func IPConvert(ip string, mode IPBracketMode) string {
	if strings.Index(ip, ".") > 0 {
		return ip
	}
	switch mode {
	case IPV6BracketNone:
		return strings.Replace(strings.Replace(ip, "[", "", -1), "]", "", -1)
	case IPV6Bracket:
		if strings.Index(ip, "[") >= 0 {
			return ip
		}
		return fmt.Sprintf("[%s]", ip)
	}
	return ip
}

// AddrConvert 含有Port的地址格式化：IPV4无变化，IPV6根据mode进行修正（有无方括号）
func AddrConvert(addr string, mode IPBracketMode) string {
	if strings.Index(addr, ".") > 0 {
		return addr
	}
	portIndex := strings.LastIndex(addr, ":")
	if portIndex <= 0 {
		return addr
	}
	ip := addr[:portIndex]
	port := ""
	if portIndex < (len(addr) - 1) {
		port = addr[portIndex+1:]
	}
	ip = IPConvert(ip, mode)
	return fmt.Sprintf("%s:%s", ip, port)
}

// SplitAddrConvert 使用逗号分隔的多个地址格式化
func SplitAddrConvert(addr string, mode IPBracketMode) string {
	ss := strings.Split(addr, ",")
	for i, s := range ss {
		ss[i] = AddrConvert(s, mode)
	}
	return strings.Join(ss, ",")
}
