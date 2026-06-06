package conf

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func MustLoad(file string, obj interface{}) {
	if err := Parse(file, obj); err != nil {
		panic(err)
	}
}

func Parse(file string, obj interface{}) error {
	v := viper.New()
	v.SetConfigFile(file)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file error: %v", err)
	}

	v.AutomaticEnv()
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := v.Unmarshal(obj); err != nil {
		return fmt.Errorf("unmarshal config file error: %v", err)
	}

	return nil
}
