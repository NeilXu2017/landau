package config

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/NeilXu2017/landau/helper"
	"gopkg.in/yaml.v2"
)

const (
	configEnvironmentTagName = "env"
)

// LoadJSONConfig 解析JSON格式文件，从环境变量读取配置
func LoadJSONConfig(configContentBytes *[]byte, appConfig interface{}) error {
	if err := json.Unmarshal(*configContentBytes, appConfig); err != nil {
		return err
	}
	v := reflect.ValueOf(appConfig)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	t := v.Type()
	if t.Kind() == reflect.Struct {
		loadFromEnvironment(v, t)
	}
	return nil
}

// LoadJSONConfig2 解析JSON格式文件，从环境变量读取配置
func LoadJSONConfig2(configFilePath string, appConfig interface{}) error {
	configFilePath = helper.ConvertAbsolutePath(configFilePath)
	configContentBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return err
	}
	return LoadJSONConfig(&configContentBytes, appConfig)
}

// LoadYamlConfig 解析YAML格式文件，从环境变量读取配置
func LoadYamlConfig(configContentBytes *[]byte, appConfig interface{}) error {
	if err := yaml.Unmarshal(*configContentBytes, appConfig); err != nil {
		return err
	}
	v := reflect.ValueOf(appConfig)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	t := v.Type()
	if t.Kind() == reflect.Struct {
		loadFromEnvironment(v, t)
	}
	return nil
}

// LoadYamlConfig2 解析YAML格式文件，从环境变量读取配置
func LoadYamlConfig2(configFilePath string, appConfig interface{}) error {
	configFilePath = helper.ConvertAbsolutePath(configFilePath)
	configContentBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return err
	}
	return LoadYamlConfig(&configContentBytes, appConfig)
}

func loadFromEnvironment(v reflect.Value, t reflect.Type) {
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		ft := fv.Type()
		switch ft.Kind() {
		case reflect.Struct:
			loadFromEnvironment(fv, ft)
		default:
			if envName, ok := t.Field(i).Tag.Lookup(configEnvironmentTagName); ok && envName != "" && fv.CanSet() {
				if val := os.Getenv(envName); val != "" {
					setFieldValue(fv, fv.Kind(), val)
				}
			}
		}
	}
}

func setFieldValue(v reflect.Value, t reflect.Kind, val string) {
	switch t {
	case reflect.String:
		v.SetString(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := strconv.Atoi(val); err == nil {
			v.SetInt(int64(i))
		}
	case reflect.Uint:
		if i, err := strconv.Atoi(val); err == nil {
			v.SetUint(uint64(i))
		}
	case reflect.Bool:
		bv := strings.ToLower(val)
		switch bv {
		case "1", "true":
			v.SetBool(true)
		case "0", "false":
			v.SetBool(false)
		default:
			if bv != "" {
				v.SetBool(true)
			}
		}
	default:
	}
}
