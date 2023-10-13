package protocol

type (
	//GateWayResponse API网关JSON输出的基本属性
	GateWayResponse struct {
		Action  string `json:"Action"`
		RetCode int    `json:"RetCode"`
		Message string `json:"Message"`
	}
)
