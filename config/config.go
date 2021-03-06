/**
 * Copyright (c) 2015 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/trustedanalytics/app-launcher-helper/service"
	"encoding/json"
)

type Config struct {
	TokenKeyUrl       string `envconfig:"TOKEN_KEY_URL"`
	ApiUrl            string `envconfig:"API_URL"`
	ServiceLabel      string `envconfig:"SERVICE_NAME"`
	VcapServicesRaw   string `envconfig:"VCAP_SERVICES"`
	VcapServices      service.VcapServices
	ServiceCatalogUrl string
}

func NewConfig() *Config {
	var c Config
	envconfig.Process("dashboard", &c)
	SetConfigVarsFromVcapServices(&c)
	return &c
}

func SetConfigVarsFromVcapServices(c *Config) {
	if err := json.Unmarshal([]byte(c.VcapServicesRaw), &c.VcapServices); err != nil {
		panic(err)
	}
	for _, s := range c.VcapServices.UpsisList {
		if (s.UpsiName == "servicecatalog") {
			c.ServiceCatalogUrl = s.UpsiCredentials.Host
		}
	}
}
