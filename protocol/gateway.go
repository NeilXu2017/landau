package protocol

type (
	//GateWayResponse API网关JSON输出的基本属性
	GateWayResponse struct {
		Action  string `json:"Action"`
		Code    int    `json:"Code"`
		Message string `json:"Message"`
	}
)
