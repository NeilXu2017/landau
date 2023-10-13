package test

import (
	"fmt"
	"reflect"

	"github.com/NeilXu2017/landau/data"
)

// CreateIndex create index
func CreateIndex(addr string) {
	mapping := `
	{
		"mappings": {
			"properties": {
				"ID": { "type": "keyword" },
				"ResourceID": { "type": "keyword" },
				"Name": { "type": "text"},
				"ResourceTypeName": { "type": "text" },
				"BusinessGroupID": { "type": "text" },
				"BusinessGroupName": { "type": "text" },
				"ProjectName": { "type": "text" },
				"ZoneName": { "type": "text" },
				"RegionName": { "type": "text" },
				"Bandwidth": { "type": "text" },
				"Config": { "type": "text" },
				"Status": { "type": "text" },
				"PrivateIP": { "type": "text" },
				"PublicIP": { "type": "text" },
				"ResourceType": { "type": "integer" },
				"ProjectID": { "type": "integer" },
				"ZoneID": { "type": "integer" },
				"RegionID": { "type": "integer" },
				"CompanyID": { "type": "integer" },
				"ResourceStatus": { "type": "integer" },
				"create_time":{"type":"integer"},
				"update_time":{"type":"integer"},
				"BindResources": { 
					"properties":{
						"ID": { "type": "keyword" },
						"ResourceID": { "type": "keyword" },
						"Name": { "type": "text" },
						"ResourceType": { "type": "integer" },
						"ProjectID": { "type": "integer" },
						"ZoneID": { "type": "integer" },
						"RegionID": { "type": "integer" },
						"CompanyID": { "type": "integer" },
						"ResourceStatus": { "type": "integer" }
					}					
				}
			}
		}
	}	
	`
	indexName := "resource"
	esClient, err := data.NewElasticSearchClient(data.SetElasticSearchURL(addr))
	if err != nil {
		fmt.Println(err)
		return
	}
	existed, err := esClient.IndexExist(indexName)
	if err != nil {
		fmt.Println(err)
		return
	}
	if existed {
		r, err := esClient.UpdateIndexMapping(indexName, mapping)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("UpdateIndexMapping:%v", r)
		return
	}
	result, err := esClient.CreateIndex(indexName, mapping)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("CreateIndex:%v", result)
}

func DeleteIndex(addr string) {
	esClient, err := data.NewElasticSearchClient(data.SetElasticSearchURL(addr))
	if err != nil {
		fmt.Println(err)
		return
	}
	_ = esClient.DeleteIndex("resource")
}

func putESData(strJSON, ID string, addr string, routingKey string) {
	esClient, err := data.NewElasticSearchClient(data.SetElasticSearchURL(addr), data.SetElasticSearchRoutingKey(routingKey))
	if err != nil {
		fmt.Println(err)
		return
	}
	existed, err := esClient.IndexExist("resource")
	if err != nil {
		fmt.Println(err)
		return
	}
	if !existed {
		fmt.Println("resource index not existed")
		return
	}
	putResult, err := esClient.PutString("resource", ID, strJSON)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%v", putResult)
}

func PutESData(addr string) {
	strJSON := `
	{
		"ID": "eip-g25e22k3",
		"ResourceID": "eip-g25e22k3",
		"Name": "EIP",
		"ResourceType": 10,
		"ResourceTypeName": "net",
		"BusinessGroupID": "business_id-s0a0zf",
		"BusinessGroupName": "美丽新世界",
		"ProjectID": 200000724,
		"ProjectName": "默认项目",
		"ZoneID": 666888,
		"ZoneName": "pre",
		"RegionID": 666888,
		"RegionName": "pre",
		"CompanyID": 200000424,
		"ResourceStatus": 1,
		"Bandwidth": "20Mb",
		"Config": "N/A",
		"PrivateIP": ["192.168.5.3", "172.28.3.6"],
		"PublicIP": ["211.19.38.210"],
		"Status": "Normal",
		"create_time":1565840030,
		"update_time":1565840030,
		"BindResources": [{
				"ID": "8bc3u258-5e01-4e43-931d-7d02fa9780adGgg",
				"ResourceID": "8bb5b258-5e01-4e43-931d-7d02fa9780adGgg",
				"Name": "",
				"ResourceType": 1,
				"ResourceTypeName": "bmc",
				"ProjectID": 200000979,
				"ZoneID": 666888,
				"RegionID": 666888,
				"CompanyID": 200000575,
				"ResourceStatus": 1
		}]
	}
	`
	putESData(strJSON, "eip-g25e22k3", addr, "200000424")
	s2 := `
	{
		"ID": "eip-c34yk43c",
		"ResourceID": "eip-c34yk43c",
		"Name": "广州对外EIP",
		"ResourceType": 10,
		"ResourceTypeName": "net",
		"BusinessGroupID": "business_id-rUPMyNDZg",
		"BusinessGroupName": "世界",
		"ProjectID": 200000724,
		"ProjectName": "默认项目",
		"ZoneID": 666888,
		"ZoneName": "pre",
		"RegionID": 666888,
		"RegionName": "pre",
		"CompanyID": 200000424,
		"ResourceStatus": 1,
		"Bandwidth": "1Mb",
		"Config": "高速双线",
		"PrivateIP": ["192.168.10.1", "172.28.10.1"],
		"PublicIP": ["156.169.250.167"],
		"Status": "Normal",
		"create_time":1565840030,
		"update_time":1565840030,
		"BindResources": [{
			"ID": "8b4tb258-5e01-4e43-931d-7d02fa9780adGgg",
			"ResourceID": "8b4tb258-5e01-4e43-931d-7d02fa9780adGgg",
			"Name": "",
			"ResourceType": 1,
			"ResourceTypeName": "host",
			"ProjectID": 200000979,
			"ZoneID": 666888,
			"RegionID": 666888,
			"CompanyID": 200000575,
			"ResourceStatus": 1
		}]
	}	
	`
	putESData(s2, "eip-ck45tk43c", addr, "200000424")
	s3 := `
	{
		"ID": "eip-1lk4y6iel",
		"ResourceID": "eip-1lk4y6iel",
		"Name": "雄霸天下组服务IP",
		"ResourceType": 10,
		"ResourceTypeName": "net",
		"BusinessGroupID": "business_id-gtz2td",
		"BusinessGroupName": "风云",
		"ProjectID": 200001453,
		"ProjectName": "风云汇",
		"ZoneID": 666888,
		"ZoneName": "pre",
		"RegionID": 666888,
		"RegionName": "pre",
		"CompanyID": 200000424,
		"ResourceStatus": 1,
		"Bandwidth": "4Mb",
		"Config": "极致高速",
		"PrivateIP": ["192.168.10.2","192.168.10.10"],
		"PublicIP": ["211.169.250.186"],
		"create_time":1565840030,
		"update_time":1565840030,
		"Status": "Normal"
	}
	`
	putESData(s3, "eip-1lk5y7iel", addr, "200000424")
}

func GetESData(addr string) {
	esClient, err := data.NewElasticSearchClient(data.SetElasticSearchURL(addr))
	if err != nil {
		fmt.Println(err)
		return
	}
	result, count, err := esClient.GetByID("resource", "eip-1lk5f4iel")
	if err != nil {
		fmt.Println(err)
		return
	}
	if count == 0 {
		fmt.Println("Not found")
		return
	}
	fmt.Println(string(result.Source))
}

func Search(addr string) {
	esClient, err := data.NewElasticSearchClient(data.SetElasticSearchURL(addr), data.SetElasticSearchRoutingKey("200000424"))
	if err != nil {
		fmt.Println(err)
		return
	}
	var queries []data.ESSimpleQuery
	q := data.ESSimpleQuery{
		Type:       data.ESMatchPhrase,
		Name:       "Config",
		Value:      "高速",
		ClauseType: data.ESSearchBoolMust,
	}
	queries = append(queries, q)
	q1 := data.ESSimpleQuery{
		Type:       data.ESTermQuery,
		Name:       "ProjectID",
		Value:      200000724,
		ClauseType: data.ESSearchBoolFilter,
	}
	queries = append(queries, q1)
	result, count, err := esClient.Search("resource", queries, 0, 1000, "", true)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Found Count:%d\n", count)
	elements := make(map[string]interface{})
	for _, item := range result.Each(reflect.TypeOf(elements)) {
		t := item.(map[string]interface{})
		fmt.Printf("%v\n", t)
	}
}
