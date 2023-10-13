package test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/NeilXu2017/landau/data"
)

type (
	addrStruct struct {
		Street        string      `json:"street,default=吴中路"`
		Building      int         `json:"building,default=2"`
		Room          int         `json:"room"`
		ChildName     []string    `json:"child_name"`
		ChildAge      []int       `json:"child_age"`
		ChildBirthDay []time.Time `json:"child_birthday" time_format:"2006-01-02 15:04:05"`
	}
	destStruct struct {
		Name     string       `json:"name"`
		Age      int          `json:"age"`
		City     string       `json:"city,default=Shanghai"`
		Address  []addrStruct `json:"address"`
		LoveGame []string     `json:"love_game"`
		Fee      []int        `json:"fee"`
		BirthDay []time.Time  `json:"birthday" time_format:"2006-01-02 15:04:05"`
	}
	BaseRequest struct {
		Action      string
		RequestUUID string `json:"request_uuid"`
	}
	_PageQuery struct {
		Offset int
		Limit  int
	}
	_CheckRequest struct {
		BaseRequest
		Name string `json:"name"`
		Age  int
		_PageQuery
		Birthday string      `json:"birthday"`
		Friend   _FriendInfo `json:"friend_info"`
	}
	_FriendInfo struct {
		Home string
		Year string
	}
)

// CheckJSONUnmarshal 测试 JSONUnmarshal 方法
func CheckJSONUnmarshal() {
	str := `
	{
		"name":"张三",
		"age":18,
		"address":{
			"room":"1202",
			"child_name":["阿猫","阿狗","赖皮"],
			"child_age":["15",20,"25"],
			"child_birthday":["2019-01-01 00:00:00","2017-03-21 10:16:00"]
		},
		"love_game":["怪物猎人"],
		"fee":["30",50,"100"],
		"birthday":"2019-01-12 12:35:12"
	}
	`
	dest := destStruct{}
	err := data.JSONUnmarshal([]byte(str), &dest)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(dest)
}

func CheckInnerJSONUnmarshal() {
	str := `
	{
		"name":"张三",
		"Age":"18",
		"Offset": 12,
		"birthday": "2003-01-02",
		"Action":"IGetClassmate",
		"Limit":25,
		"request_uuid":"a5cd-1234-h7il-6632",
		"friend_info": {
			"Home":"Shanghai",
			"Year":"2001"
		}
	}
	`
	dest := _CheckRequest{}
	err := data.JSONUnmarshal([]byte(str), &dest)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(dest)

	dest2 := _CheckRequest{}
	_ = json.Unmarshal([]byte(str), &dest2)
	fmt.Print(dest2)

	dest3 := make(map[string]interface{})
	_ = json.Unmarshal([]byte(str), &dest3)
	fmt.Print(dest3)
}
