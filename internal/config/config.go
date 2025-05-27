// Package config defines the necessary types to configure the application.
// An example config file config.yaml is provided in the repository.
package config

import (
	"time"

	"github.com/openkcm/common-sdk/pkg/commoncfg"
)

type CheckType string

const (
	ContainsCheckType          CheckType = "Contains"
	RegularExpressionCheckType CheckType = "RegularExpression"
	SuffixCheckType            CheckType = "Suffix"
	PrefixCheckType            CheckType = "Prefix"
	EqualCheckType             CheckType = "Equal"
)

type SourceType string

const (
	ResponseBodySourceType   SourceType = "ResponseBody"
	ResponseStatusSourceType SourceType = "ResponseStatus"
)

type Config struct {
	commoncfg.BaseConfig `mapstructure:",squash"`

	Server      Server      `yaml:"server"`
	Healthcheck Healthcheck `yaml:"healthcheck"`
	Versions    Versions    `yaml:"versions"`
}

type Server struct {
	// HTTP.Address is the address to listen on for HTTP requests
	Address         string        `yaml:"address" default:":8080"`
	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" default:"5s"`
}

type Versions struct {
	Enabled   bool               `yaml:"enabled" default:"false"`
	Endpoint  string             `yaml:"endpoint" default:"/versions"`
	Resources []*ServiceResource `yaml:"resources" default:"[]"`
}

type ServiceResource struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type Healthcheck struct {
	Enabled    bool       `yaml:"enabled" default:"false"`
	Endpoint   string     `yaml:"endpoint" default:"/healthz"`
	Cluster    Cluster    `yaml:"cluster"`
	Kubernetes Kubernetes `yaml:"kubernetes"`
	Linkerd    Linkerd    `yaml:"linkerd"`
}

type Cluster struct {
	Enabled   bool              `yaml:"enabled"`
	Tag       string            `yaml:"tag" default:"cluster"`
	Resources []ClusterResource `yaml:"resources"`
}

type ClusterResource struct {
	Name   string  `yaml:"name"`
	URL    string  `yaml:"url"`
	Checks []Check `yaml:"checks"`
}

type Linkerd struct {
	Enabled               bool     `yaml:"enabled"`
	Tag                   string   `yaml:"tag" default:"linkerd"`
	ControlPlaneNamespace string   `yaml:"controlPlaneNamespace" default:"linkerd"`
	DataPlaneNamespace    string   `yaml:"dataPlaneNamespace" default:"linkerd"`
	CNINamespace          string   `yaml:"cniNamespace" default:"linkerd-cni"`
	RetryDeadline         int      `yaml:"retryDeadline" default:"300"`
	CNIEnabled            bool     `yaml:"cniEnabled"`
	Output                string   `yaml:"output" default:"short"`
	Checks                []string `yaml:"checks"`
}

type Kubernetes struct {
	Enabled   bool                 `yaml:"enabled"`
	Tag       string               `yaml:"tag" default:"kubernetes"`
	Resources []KubernetesResource `yaml:"resources"`
}
type KubernetesResource struct {
	Name   string  `yaml:"name"`
	URL    string  `yaml:"url"`
	Checks []Check `yaml:"checks"`
}

type Check struct {
	Type   CheckType  `yaml:"type" default:"Contains"`
	Source SourceType `yaml:"source" default:"ResponseBody"`
	Value  string     `yaml:"value"`
}
