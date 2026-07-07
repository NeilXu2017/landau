package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var (
	byteSizeMap map[reflect.Kind]int
)

const (
	_tagNameJSON            = "json"
	_tagNameTimeFormat      = "time_format"
	_tagNameTimeUTC         = "time_utc"
	_tagNameTimeLocation    = "time_location"
	_tagJSONDefaultProperty = "default"
	_tagNameSupportGWArray  = "gw_array"
)

func init() {
	byteSizeMap = make(map[reflect.Kind]int)
	byteSizeMap[reflect.Int] = 0
	byteSizeMap[reflect.Int8] = 8
	byteSizeMap[reflect.Int16] = 16
	byteSizeMap[reflect.Int32] = 32
	byteSizeMap[reflect.Int64] = 64
	byteSizeMap[reflect.Uint] = 0
	byteSizeMap[reflect.Uint8] = 8
	byteSizeMap[reflect.Uint16] = 16
	byteSizeMap[reflect.Uint32] = 32
	byteSizeMap[reflect.Uint64] = 64
	byteSizeMap[reflect.Float32] = 32
	byteSizeMap[reflect.Float64] = 64
}

// JSONUnmarshal 增强JSON Unmarshal 方法：支持字符与数值型互转和[]形式
// 示例:
//
//	[]string	可接受	xxx:["Tom","Jerry","Stevin",123,123.5] 或者  xxx:"Monica" 或者 xxx:189
//	[]int	可接受	xxx:[1,2,"3"] 或者 xxx:"1" 或者 xxx:2
//	以上形式也允许嵌套struct 或者 []struct
func JSONUnmarshal(b []byte, ptr interface{}) error {
	m := make(map[string]interface{})
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	return jsonUnmarshalFromMap(ptr, m)
}

// 不区分大小写，找到对应的存在map 中的 Key
func getCaseInsensitiveKey(inputFieldName string, m map[string]interface{}) string {
	lowerFieldName := strings.ToLower(inputFieldName)
	for k := range m {
		if lowerFieldName == strings.ToLower(k) {
			return k
		}
	}
	return inputFieldName
}

func _prepareGWArrayParameter(paramName string, m map[string]interface{}) map[string]interface{} {
	regExp := regexp.MustCompile(fmt.Sprintf(`%s\.\d{1,10}`, paramName))
	n := make(map[string]interface{})
	var paramValue []interface{}
	for k, v := range m {
		nv := v
		if regExp.MatchString(k) {
			paramValue = append(paramValue, nv)
		} else {
			n[k] = nv
		}
	}
	if len(paramValue) == 0 {
		if v, ok := m[paramName]; ok {
			n[paramName] = v
		}
	} else {
		n[paramName] = paramValue
	}
	return n
}

func jsonUnmarshalFromMap(ptr interface{}, m map[string]interface{}) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		structFieldKind := structField.Kind()
		if !structField.CanSet() {
			if structFieldKind == reflect.Struct && typeField.Anonymous { //非可导出成员
				value2 := reflect.New(structField.Type()).Elem()
				if err := jsonUnmarshalFromMap(value2.Addr().Interface(), m); err != nil {
					return err
				}
				reflect.NewAt(structField.Type(), unsafe.Pointer(structField.UnsafeAddr())).Elem().Set(value2)
			}
			continue
		}
		inputFieldName := typeField.Tag.Get(_tagNameJSON)
		inputFieldNameList := strings.Split(inputFieldName, ",")
		inputFieldName = inputFieldNameList[0]
		if inputFieldName == "-" {
			continue
		}
		if inputFieldName == "" {
			inputFieldName = typeField.Name
		}
		supportGWArray := typeField.Tag.Get(_tagNameSupportGWArray)
		if supportGWArray == "true" {
			m = _prepareGWArrayParameter(inputFieldName, m)
		}

		if structFieldKind == reflect.Ptr {
			if !structField.Elem().IsValid() {
				structField.Set(reflect.New(structField.Type().Elem()))
			}
			structField = structField.Elem()
			structFieldKind = structField.Kind()
		}
		if structFieldKind == reflect.Struct {
			inputFieldNameMapKey := getCaseInsensitiveKey(inputFieldName, m)
			if typeField.Anonymous {
				if err := jsonUnmarshalFromMap(structField.Addr().Interface(), m); err != nil {
					return err
				}
			} else {
				if v, ok := m[inputFieldNameMapKey]; ok {
					if _, isTime := structField.Interface().(time.Time); isTime {
						if err := setTimePropertyValue(typeField, v, structField); err != nil {
							return err
						}
						continue
					}
					if nm, ok := v.(map[string]interface{}); ok {
						if err := jsonUnmarshalFromMap(structField.Addr().Interface(), nm); err != nil {
							return err
						}
					}
				}
			}
			continue
		}
		var defaultValue string
		if len(inputFieldNameList) > 1 {
			defaultList := strings.SplitN(inputFieldNameList[1], "=", 2)
			if defaultList[0] == _tagJSONDefaultProperty {
				defaultValue = defaultList[1]
			}
		}
		inputFieldNameMapKey := getCaseInsensitiveKey(inputFieldName, m)
		inputValue, exists := m[inputFieldNameMapKey]
		if !exists {
			if defaultValue == "" {
				continue
			}
			inputValue = defaultValue
		}
		if structFieldKind == reflect.Slice {
			sliceValues, numElems, err := getSliceValue(inputValue)
			if err != nil {
				return err
			}
			if numElems > 0 {
				sliceOf := structField.Type().Elem().Kind()
				slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
				for i := 0; i < numElems; i++ {
					if sliceOf == reflect.Struct {
						if _, isTime := slice.Index(i).Interface().(time.Time); isTime {
							if err := setTimePropertyValue(typeField, sliceValues[i], slice.Index(i)); err != nil {
								return err
							}
							continue
						}
						sliceMap, ok := sliceValues[i].(map[string]interface{})
						if !ok {
							return fmt.Errorf("invalid data:%v", sliceValues[i])
						}
						if err := jsonUnmarshalFromMap(slice.Index(i).Addr().Interface(), sliceMap); err != nil {
							return err
						}
					} else {
						if err := setPropertyValue(sliceOf, sliceValues[i], slice.Index(i)); err != nil {
							return err
						}
					}
				}
				val.Field(i).Set(slice)
			}
			continue
		}
		if err := setPropertyValue(typeField.Type.Kind(), inputValue, structField); err != nil {
			return err
		}
	}
	return nil
}

func getSliceValue(val interface{}) ([]interface{}, int, error) {
	var ret []interface{}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i, j := 0, v.Len(); i < j; i++ {
			ret = append(ret, v.Index(i).Interface())
		}
	default:
		ret = append(ret, v.Interface())
	}
	return ret, len(ret), nil
}

func setPropertyValue(valueKind reflect.Kind, v interface{}, value reflect.Value) error {
	switch valueKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := toInt(v, valueKind)
		if err != nil {
			return err
		}
		value.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := toUInt(v, valueKind)
		if err != nil {
			return err
		}
		value.SetUint(val)
	case reflect.Float32, reflect.Float64:
		val, err := toFloat(v, valueKind)
		if err != nil {
			return err
		}
		value.SetFloat(val)
	case reflect.String:
		val, err := toString(v)
		if err != nil {
			return err
		}
		value.SetString(val)
	case reflect.Bool:
		val, err := toBool(v)
		if err != nil {
			return err
		}
		value.SetBool(val)
	case reflect.Ptr:
		if !value.Elem().IsValid() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		structFieldElem := value.Elem()
		return setPropertyValue(structFieldElem.Kind(), v, structFieldElem)
	default:
		return errors.New("unknown type")
	}
	return nil
}

func setTimePropertyValue(structField reflect.StructField, v interface{}, value reflect.Value) error {
	val, ok := v.(string)
	if !ok {
		return fmt.Errorf("not string:%v", v)
	}
	timeFormat := structField.Tag.Get(_tagNameTimeFormat)
	if timeFormat == "" {
		return errors.New("blank time format")
	}
	if val == "" {
		value.Set(reflect.ValueOf(time.Time{}))
		return nil
	}
	l := time.Local
	if isUTC, _ := strconv.ParseBool(structField.Tag.Get(_tagNameTimeUTC)); isUTC {
		l = time.UTC
	}
	if locTag := structField.Tag.Get(_tagNameTimeLocation); locTag != "" {
		loc, err := time.LoadLocation(locTag)
		if err != nil {
			return err
		}
		l = loc
	}
	t, err := time.ParseInLocation(timeFormat, val, l)
	if err != nil {
		return err
	}
	value.Set(reflect.ValueOf(t))
	return nil
}

func toInt(v interface{}, valueKind reflect.Kind) (int64, error) {
	switch nv := v.(type) {
	case int:
		return int64(nv), nil
	case int8:
		return int64(nv), nil
	case int16:
		return int64(nv), nil
	case int32:
		return int64(nv), nil
	case int64:
		return nv, nil
	case uint:
		return int64(nv), nil
	case uint8:
		return int64(nv), nil
	case uint16:
		return int64(nv), nil
	case uint32:
		return int64(nv), nil
	case uint64:
		return int64(nv), nil
	case float32:
		return int64(nv), nil
	case float64:
		return int64(nv), nil
	case string:
		val := v.(string)
		intVal, err := strconv.ParseInt(val, 10, byteSizeMap[valueKind])
		return intVal, err
	}
	return 0, fmt.Errorf("unsupport to int %v", v)
}

func toUInt(v interface{}, valueKind reflect.Kind) (uint64, error) {
	switch nv := v.(type) {
	case int:
		return uint64(nv), nil
	case int8:
		return uint64(nv), nil
	case int16:
		return uint64(nv), nil
	case int32:
		return uint64(nv), nil
	case int64:
		return uint64(nv), nil
	case uint:
		return uint64(nv), nil
	case uint8:
		return uint64(nv), nil
	case uint16:
		return uint64(nv), nil
	case uint32:
		return uint64(nv), nil
	case uint64:
		return nv, nil
	case float32:
		return uint64(nv), nil
	case float64:
		return uint64(nv), nil
	case string:
		val := v.(string)
		intVal, err := strconv.ParseUint(val, 10, byteSizeMap[valueKind])
		return intVal, err
	}
	return 0, fmt.Errorf("unsupport to uint %v", v)
}

func toFloat(v interface{}, valueKind reflect.Kind) (float64, error) {
	switch nv := v.(type) {
	case int:
		return float64(nv), nil
	case int8:
		return float64(nv), nil
	case int16:
		return float64(nv), nil
	case int32:
		return float64(nv), nil
	case int64:
		return float64(nv), nil
	case uint:
		return float64(nv), nil
	case uint8:
		return float64(nv), nil
	case uint16:
		return float64(nv), nil
	case uint32:
		return float64(nv), nil
	case uint64:
		return float64(nv), nil
	case float32:
		return float64(nv), nil
	case float64:
		return nv, nil
	case string:
		val := v.(string)
		floatVal, err := strconv.ParseFloat(val, byteSizeMap[valueKind])
		return floatVal, err
	}
	return 0, fmt.Errorf("unsupport to float %v", v)
}

func toString(v interface{}) (string, error) {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%g", v), nil
	case string:
		return v.(string), nil
	}
	return "", fmt.Errorf("unsupport to string %v", v)
}

func toBool(v interface{}) (bool, error) {
	str := ""
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		str = fmt.Sprintf("%d", v)
	case float32, float64:
		str = fmt.Sprintf("%g", v)
	case string:
		str = v.(string)
	default:
		str = fmt.Sprintf("%v", v)
	}
	return strconv.ParseBool(str)
}
