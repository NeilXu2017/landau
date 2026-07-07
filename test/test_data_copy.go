package test

import (
	"fmt"

	"github.com/NeilXu2017/landau/util"
)

type (
	testSource struct {
		Name      string
		Age       int
		IsDeleted bool
		Member    map[string]string
		Inner     testSourceInner
		Loves     []string
		Like      map[string]testSourceInner
	}
	testSourceInner struct {
		CreateTime int
		Locations  []string
	}

	testDest struct {
		Name      string
		Age       int
		Member    map[string]string
		Inner     testSourceInner
		Loves     []string
		Like      map[string]testSourceInner
		IsDeleted bool
		Other     string
	}
)

// CheckDataCopy 测试结构体COPY
func CheckDataCopy() {
	fmt.Println("CHECK DATA COPY")
	src := testSource{
		Name:      "我是 who?",
		Age:       19,
		IsDeleted: true,
		Member:    map[string]string{"Key1": "Value1", "Key2": "Value2"},
		Inner:     testSourceInner{CreateTime: 1900, Locations: []string{"A", "B", "C", "D"}},
		Loves:     []string{"AA", "BB", "CC"},
		Like: map[string]testSourceInner{
			"K1": {
				CreateTime: 1,
				Locations:  []string{"K1A", "K1B"},
			},
			"K2": {
				CreateTime: 2,
				Locations:  []string{"K2A", "K2B"},
			},
		},
	}
	fmt.Printf("%v\n", src)
	dst := testDest{}
	err := util.Copy(&dst, src)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	fmt.Printf("Name:%s\n", dst.Name)
	fmt.Printf("Age:%d\n", dst.Age)
	fmt.Printf("Member:%v\n", dst.Member)
	fmt.Printf("Inner:%v\n", dst.Inner)
	fmt.Printf("Loves:%v\n", dst.Loves)
	fmt.Printf("Like:%v\n", dst.Like)
	fmt.Printf("IsDeleted:%v\n", dst.IsDeleted)
	fmt.Printf("Other:%v\n", dst.Other)
}
