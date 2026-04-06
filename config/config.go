package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/spf13/viper"
)

type Configuration struct {
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBName     string `mapstructure:"DB_NAME"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBUsername string `mapstructure:"DB_USER"`

	RedisUrl  string `mapstructure:"REDIS_URL"`
	RedisPass string `mapstructure:"REDIS_PASS"`

	Secret string `mapstructure:"SECRET"`
}

var GlobalConfig *Configuration

func LoadConfig(path string) (*Configuration, error) {
	c := Configuration{}

	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	// 🔥 ignore error → Render has no config file
	_ = viper.ReadInConfig()

	// try env first (Render)
	envCfg, _ := loadFromEnv()
	if envCfg != nil {
		GlobalConfig = envCfg
		return GlobalConfig, nil
	}

	// fallback to file (local)
	if err := viper.Unmarshal(&c); err != nil {
		return nil, err
	}

	GlobalConfig = &c
	return GlobalConfig, nil
}

func loadFromEnv() (*Configuration, error) {
	c := Configuration{}

	v := reflect.ValueOf(&c).Elem()
	t := v.Type()

	found := false

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		envVar := field.Tag.Get("mapstructure")
		if envVar == "" {
			continue
		}

		envValue := os.Getenv(envVar)
		if envValue == "" {
			continue
		}

		found = true

		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(envValue)
		case reflect.Bool:
			if val, err := strconv.ParseBool(envValue); err == nil {
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
			return nil, fmt.Errorf("unsupported field type %s for field %s", fieldValue.Kind(), field.Name)
		}
	}

	if !found {
		return nil, nil
	}

	return &c, nil
}