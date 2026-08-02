package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	ListenAddr string   `yaml:"listenAddr" env-default:"0.0.0.0:8080"`
	UserAgent  string   `yaml:"user_agent" env-default:"OpinionBot"`
	Database   Database `yaml:"database"`
}

type Database struct {
	Port     string `yaml:"port" env-default:"5432"`
	Domain   string `yaml:"domain" env-default:"localhost"`
	User     string `yaml:"user" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	DBName   string `yaml:"dbname" env-required:"true"`
}

func MustLoadConf() Config {
	var config Config

	if err := cleanenv.ReadConfig("./config.yaml", &config); err != nil {
		panic(err)
	}
	return config
}
