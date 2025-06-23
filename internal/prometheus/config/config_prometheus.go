/*
@Author : WuWeiJian
@Date : 2020-12-16 15:41
*/

package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/url"
	"time"
)

// prometheus 数据库程序的配置文件
type PrometheusConfig struct {
	GlobalConfig GlobalConfig `yaml:"global"`
	//AlertingConfig AlertingConfig  `yaml:"alerting,omitempty"`
	RuleFiles     []string         `yaml:"rule_files,omitempty"`
	ScrapeConfigs []*ScrapeConfig  `yaml:"scrape_configs,omitempty"`
	StaticConfigs []*StaticConfigs `yaml:"static_configs,omitempty"`
}

// GlobalConfig configures values that are used across other configuration
// objects.
type GlobalConfig struct {
	// How frequently to scrape targets by default.
	ScrapeInterval time.Duration `yaml:"scrape_interval,omitempty"`
	// The default timeout when scraping targets.
	ScrapeTimeout time.Duration `yaml:"scrape_timeout,omitempty"`
	// How frequently to evaluate rules by default.
	EvaluationInterval time.Duration `yaml:"evaluation_interval,omitempty"`
	// File to which PromQL queries are logged.
	QueryLogFile string `yaml:"query_log_file,omitempty"`
	// The labels to add to any timeseries that this Prometheus instance scrapes.
	//ExternalLabels labels.Labels `yaml:"external_labels,omitempty"`
}

// ScrapeConfig configures a scraping unit for Prometheus.
type ScrapeConfig struct {
	// The job name to which the job label is set by default.
	JobName string `yaml:"job_name"`
	// Indicator whether the scraped metrics should remain unmodified.
	HonorLabels bool `yaml:"honor_labels,omitempty"`
	// Indicator whether the scraped timestamps should be respected.
	HonorTimestamps bool `yaml:"honor_timestamps"`
	// A set of query parameters with which the target is scraped.
	Params url.Values `yaml:"params,omitempty"`
	// How frequently to scrape the targets of this scrape config.
	ScrapeInterval time.Duration `yaml:"scrape_interval,omitempty"`
	// The timeout for scraping targets of this config.
	ScrapeTimeout time.Duration `yaml:"scrape_timeout,omitempty"`
	// The HTTP resource path on which to fetch metrics from targets.
	MetricsPath string `yaml:"metrics_path,omitempty"`
	// The URL scheme with which to fetch metrics from targets.
	Scheme string `yaml:"scheme,omitempty"`
	// More than this many samples post metric-relabeling will cause the scrape to fail.
	SampleLimit uint `yaml:"sample_limit,omitempty"`
	// More than this many targets after the target relabeling will cause the
	// scrapes to fail.
	TargetLimit   uint             `yaml:"target_limit,omitempty"`
	FileSdConfigs []*FileSdConfigs `yaml:"file_sd_configs,omitempty"`
}

type FileSdConfigs struct {
	Files           []string      `yaml:"files,omitempty"`
	RefreshInterval time.Duration `yaml:"refresh_interval,omitempty"`
}

type StaticConfigs struct {
	Targets []string `yaml:"targets,omitempty"`
}

func NewPrometheusConfig() *PrometheusConfig {
	global := GlobalConfig{
		ScrapeInterval:     time.Second * 30,
		EvaluationInterval: time.Second * 30,
	}
	fileSd := &FileSdConfigs{Files: []string{"conf/node.targets.json"}}
	scrape := &ScrapeConfig{
		JobName: "nodecontainers",
		FileSdConfigs: []*FileSdConfigs{fileSd},
	}
	static := &StaticConfigs{Targets: []string{}}
	return &PrometheusConfig{
		GlobalConfig: global,
		ScrapeConfigs: []*ScrapeConfig{scrape},
		StaticConfigs: []*StaticConfigs{static},
	}
}

func (c *PrometheusConfig) HandleConfig(port int) {
	target := fmt.Sprintf("localhost:%d", port)
	c.StaticConfigs[0].Targets = append(c.StaticConfigs[0].Targets, target)
}

func (c *PrometheusConfig) SaveTo(filename string) error {
	d, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, d, 0755)
}
