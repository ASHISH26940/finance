package config

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"

	"github.com/spf13/viper"
)


type Configuration struct{
	DBHost string `mapstructure:"DB_HOST"`
	DBPort string `mapstructure:"DB_PORT"`
	DBName string `mapstructure:"DB_NAME"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBUsername string `mapstructure:"DB_USER"`

	RedisUrl string `mapstructure:"REDIS_URL"`
	RedisPass string `mapstructure:"REDIS_PASS"`

	Secret string `mapstructure:"SECRET"`
}

var GlobalConfig *Configuration

func LoadConfig(path string)(*Configuration,error){
	if os.Getenv("FN_ENV")=="PROD"{
		c,err:=LoadConfigLoad()
		if err!=nil{
			return nil,err
		}
		GlobalConfig=c
		return GlobalConfig,nil
	}
	viper.AddConfigPath(path)
	viper.SetConfigFile("env")
	viper.SetConfigName("app")

	viper.AutomaticEnv()

	err:=viper.ReadInConfig()
	if err!=nil{
		return nil,err
	}

	err=viper.Unmarshal(&GlobalConfig)
	if err!=nil{
		return nil,err
	}

	return GlobalConfig,nil
}

func LoadConfigLoad()(*Configuration,error){
	c:=Configuration{}

	v:=reflect.ValueOf(&c).Elem()
	t:=v.Type()

	for i:=0;i<t.NumField();i++{
		field:=t.Field(i)
		fieldValue:=v.Field(i)

		if !fieldValue.CanSet(){
			continue
		}

		envVar:=field.Tag.Get("mapstructure")
		if envVar==""{
			continue
		}

		envValue:=os.Getenv(envVar)
		if envValue==""{
			log.Print("Warning:cant find this env in enviroment",envVar)
			continue
		}

		switch fieldValue.Kind(){
		case reflect.String:
			fieldValue.SetString(envValue)
		case reflect.Bool:
			if val,err:=strconv.ParseBool(envValue);err==nil{
				fieldValue.SetBool(val)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if val, err := strconv.ParseInt(envValue, 10, 64); err == nil {
				fieldValue.SetInt(val)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if val, err := strconv.ParseUint(envValue, 10, 64); err == nil {
				fieldValue.SetUint(val)
			}
		case reflect.Float32, reflect.Float64:
			if val, err := strconv.ParseFloat(envValue, 64); err == nil {
				fieldValue.SetFloat(val)
			}
		default:
			return nil,fmt.Errorf("unsupported field type %s for the feild %s",fieldValue.Kind(),field.Name)
		}
	}

	return &c,nil
}