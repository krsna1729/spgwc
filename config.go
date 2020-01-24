// Copyright 2019-2020 go-gtp authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Config is a configurations loaded from yaml.
type Config struct {
	S11Addr string `yaml:"s11_addr"`

	UESubnet string `yaml:"ue_subnet"`

	UPFs []struct {
		SxAddr  string `yaml:"sx_addr"`
		S1UAddr string `yaml:"s1u_addr"`
	} `yaml:"upfs,flow"`
}

func loadConfig(path string) (*Config, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	if err := yaml.Unmarshal(buf, c); err != nil {
		return nil, err
	}
	log.Println(c)
	return c, nil
}
