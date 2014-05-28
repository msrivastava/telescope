package main

import (
	"strings"
)

type AccessControl struct {
	ClientIP   string   `json:"ip"`
	MeterNames []string `json:"meters"`
}

func (this AccessControl) Match(addr string) bool {
	if this.ClientIP == "*" {
		return true
	}
	return this.ClientIP == strings.Split(addr, ":")[0]
}

func (this AccessControl) HasMeter(meter string) bool {
	for _, m := range this.MeterNames {
		if m == meter {
			return true
		}
	}
	return false
}
