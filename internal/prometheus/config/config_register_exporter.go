package config

import (
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"regexp"
	"strings"
)

var strReg = regexp.MustCompile(`^[A-Za-z0-9]+$`)
var numberReg = regexp.MustCompile(`^[0-9]+$`)

var JobNameList = []string{
	"node_exporter",
	"mysql_exporter",
	"mongodb_exporter",
	"redis_exporter",
	"postgres_exporter",
	"pgpool2_exporter",
	"windows_exporter",
}

type RegisterExporterConf struct {
	JobName   string
	MetricUrl string
	Tags      map[string]string
	Interval  string
	//ConsulAddr        string
	PrometheusConfDir string
	PrometheusPort    int
}

func (r *RegisterExporterConf) InitArgs() {
	if len(r.Interval) == 0 {
		r.Interval = "60s"
	}

	if len(r.PrometheusConfDir) == 0 {
		r.PrometheusConfDir = DefaultPrometheusConfDir
	}

	if r.PrometheusPort == 0 {
		r.PrometheusPort = 9090
	}
}

func (r *RegisterExporterConf) Validator() error {
	logger.Infof("验证参数\n")

	if !r.inJobNameList(r.JobName) {
		return fmt.Errorf("任务类型必须在 %s 中", strings.Join(JobNameList, ","))
	}

	if !utils.IsExists(r.PrometheusConfDir) {
		return fmt.Errorf("prometheus 的配置目录 %s 在本机不存在", r.PrometheusConfDir)
	}

	if len(r.Tags) > 0 {
		for k, v := range r.Tags {
			if !strReg.MatchString(k) || !strReg.MatchString(v) {
				return fmt.Errorf("tags 只能包含数字字母")
			}

			if numberReg.MatchString(k) {
				return fmt.Errorf("tags 的 key 不能为纯数字")
			}
		}
	}

	return nil
}

func (r *RegisterExporterConf) inJobNameList(name string) bool {
	for _, jobName := range JobNameList {
		if jobName == name {
			return true
		}
	}
	return false
}
