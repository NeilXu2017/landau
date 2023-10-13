package test

import (
	"github.com/NeilXu2017/landau/data"
	"github.com/globalsign/mgo/bson"
)

type (
	mgoRowData map[string]interface{}
)

// CheckMongoDB 测试MongoDB
func CheckMongoDB() {
	c := data.NewMongoDatabase2("172.28.39.194", 27017, "hybrid", "hybrid", "hybrid_123")
	bsonQuery := bson.M{}
	oneFieldBsonCondition := bson.M{"$gte": 0, "$lt": 1555257600}
	searchFields := make(map[string]string)
	searchFields["create_time"] = "create_time"
	searchFields["modify_time"] = "modify_time"
	searchFields["deleted_time"] = "deleted_time"
	if len(searchFields) == 1 {
		for k := range searchFields {
			bsonQuery[k] = oneFieldBsonCondition
		}
	} else {
		var allQueryFields []bson.M
		for k := range searchFields {
			v := bson.M{k: oneFieldBsonCondition}
			allQueryFields = append(allQueryFields, v)
		}
		bsonQuery["$or"] = allQueryFields
	}
	_, _ = c.Count("hybrid", "rack", bsonQuery)
	var rows []mgoRowData
	_ = c.GetPage("hybrid", "rack", bsonQuery, bson.M{}, &rows, 1, 15)
	row := mgoRowData{}
	_ = c.Get("hybrid", "rack", bsonQuery, bson.M{}, &row)
}
