package config

import (
	"encoding/json"
	"io/ioutil"
)

//Config ... a struct for Configuration
type Config struct {
	Docker        dockerConfig
	ListenAddress string `json:"listen_address"`
	DataVolume    string `json:"data_volume"`
	ReadTimeout   int    `json:"read_timeout"`
	WriteTimeout  int    `json:"write_timeout"`
	KeyLocation   string `json:"key,omitempty"`
	KeyName       string `json:"key_name,omitempty"`
}

type dockerConfig struct {
	Endpoint string `json:"endpoint"`
	Port     int    `json:"reserved_port"`
	SSLPort  int    `json:"reserved_ssl_port"`
}

//NewConfig ...
func NewConfig(path string) (Config, error) {
	cfg := Config{}
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(file, &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, err
}
