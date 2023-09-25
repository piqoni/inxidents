package main

import "time"

type Service struct {
	Name         string        `yaml:"name"`
	Endpoint     string        `yaml:"endpoint"`
	Frequency    time.Duration `yaml:"frequency"`
	ExpectedCode int           `yaml:"expectedStatusCode"`
	ExpectedBody string        `yaml:"expectedStringBody"`
}
