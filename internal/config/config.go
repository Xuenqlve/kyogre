package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/transform"
)

const (
	DefaultConfigType Type = "file"
	TypeKey                = "config-type"
	FilePathKey            = "config-path"
)

func NewConfig() (Config, error) {
	var configFile = ""
	flagSet := flag.NewFlagSet("app", flag.ContinueOnError)
	flagSet.StringVar(&configFile, "config", "", "config file")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}
	fmt.Printf("config: %s\n", configFile)
	if configFile == "" {
		return Config{}, fmt.Errorf("缺少参数 -config")
	}
	cfgData, err := transform.ConfigFromFile(configFile)
	if err != nil {
		return Config{}, errors.Trace(err)
	}
	var configType Type
	if cfgType, ok := cfgData[TypeKey]; ok {
		configType = cfgType.(Type)
	} else {
		configType = DefaultConfigType
		cfgData[FilePathKey] = configFile
	}
	config, err := GetConfig(configType)
	if err != nil {
		return Config{}, errors.Trace(err)
	}
	if err = config.Configure(cfgData); err != nil {
		return Config{}, errors.Trace(err)
	}
	cfg, err := config.Load()
	if err != nil {
		return Config{}, errors.Trace(err)
	}
	return cfg, nil
}
