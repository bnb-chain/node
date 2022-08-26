package main

import (
	"bytes"
	"fmt"
	"text/template"

	cmn "github.com/tendermint/tendermint/libs/common"
)

var configTemplate *template.Template

func init() {
	var err error
	if configTemplate, err = template.New("configFileTemplate").Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

type NodeTemplateParams struct {
	Index       int
	PortIP      int
	PortExpose1 int
	PortExpose2 int
}

type DockerComposeTemplateParams struct {
	Nodes []NodeTemplateParams
}

// WriteConfigFile renders config using the template and writes it to configFilePath.
func WriteConfigFile(configFilePath string, config *DockerComposeTemplateParams) {
	var buffer bytes.Buffer
	fmt.Printf("Writing config %+v to %s\n", config, configFilePath)
	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	cmn.MustWriteFile(configFilePath, buffer.Bytes(), 0644)
}

const defaultConfigTemplate = `version: '3'

services:
{{- range .Nodes }}

  node{{ .Index }}:
    container_name: node{{ .Index }}
    image: "binance/bnbdnode"
    restart: always
    working_dir: /bnbchaind
    command: bnbchaind start --home /bnbchaind/testnoded
    ports:
      - "{{ .PortExpose1 }}:26656"
      - "{{ .PortExpose2 }}:26657"
    volumes:
      - ./node{{ .Index }}:/bnbchaind:Z
    networks:
      localnet:
        ipv4_address: 172.20.0.{{ .PortIP }}

{{- end }}

networks:
  localnet:
    driver: bridge
    ipam:
      driver: default
      config:
      -
        subnet: 172.20.0.0/16
`
