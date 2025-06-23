package services

import (
	"crypto/md5"
	"dbup/internal/prometheus/config"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/levigross/grequests"
)

type ExporterParams struct {
	ID                string               `json:"ID"`
	Name              string               `json:"Name"`
	Tags              []string             `json:"Tags"`
	Address           string               `json:"Address"`
	Port              int                  `json:"Port"`
	Meta              exporterParamMeta    `json:"Meta"`
	EnableTagOverride bool                 `json:"EnableTagOverride"`
	Check             exporterParamCheck   `json:"Check"`
	Weights           exporterParamWeights `json:"Weights"`
}

type exporterParamMeta map[string]string

type exporterParamCheck struct {
	HTTP     string `json:"HTTP"`
	Interval string `json:"Interval"`
}

type exporterParamWeights struct {
	Passing int `json:"Passing"`
	Warning int `json:"Warning"`
}

type ExporterFileConf struct {
	Labels  exporterParamMeta `json:"labels"`
	Targets []string          `json:"targets"`
}

type registerExporter struct {
	cfg *config.RegisterExporterConf
}

func NewRegisterExporter(cfg *config.RegisterExporterConf) *registerExporter {
	r := new(registerExporter)
	r.cfg = cfg
	r.cfg.InitArgs()
	return r
}

func (r *registerExporter) Run() error {
	err := r.cfg.Validator()
	if err != nil {
		return err
	}

	uri, err := url.Parse(r.cfg.MetricUrl)
	if err != nil {
		return fmt.Errorf("解析 mertic 的 url 失败")
	}

	target := strings.Replace(r.cfg.MetricUrl, uri.Scheme+"://", "", 1)

	//meta := exporterParamMeta{}
	//if len(r.cfg.Tags) > 0 {
	//	meta = r.cfg.Tags
	//}
	//
	//check := exporterParamCheck{
	//	HTTP:     r.cfg.MetricUrl,
	//	Interval: r.cfg.Interval,
	//}
	//
	//weights := exporterParamWeights{
	//	Passing: 10,
	//	Warning: 1,
	//}

	//port, err := strconv.Atoi(uri.Port())
	//if err != nil {
	//	return fmt.Errorf("解析 consul url 失败:%w", err)
	//}

	//params := ExporterParams{
	//	ID:                uuid.NewString(),
	//	Name:              r.cfg.JobName,
	//	Tags:              []string{r.cfg.JobName},
	//	Address:           uri.Hostname(),
	//	Port:              port,
	//	Meta:              meta,
	//	EnableTagOverride: false,
	//	Check:             check,
	//	Weights:           weights,
	//}
	meta := exporterParamMeta{}
	if len(r.cfg.Tags) > 0 {
		meta = r.cfg.Tags
	}

	params := ExporterFileConf{
		Labels:  meta,
		Targets: []string{target},
	}
	fileConf := []ExporterFileConf{params}

	jBytes, _ := json.MarshalIndent(fileConf, "", " ")
	//logger.Successf("开始注册：\n%s", string(jBytes))

	//logger.Infof("开始注册 exporter 到 consul\n")
	//
	//registerUrl := fmt.Sprintf("%s/v1/agent/service/register", strings.TrimRight(r.cfg.ConsulAddr, "/"))
	//
	//_, err = grequests.Put(registerUrl, &grequests.RequestOptions{
	//	JSON:               params,
	//	InsecureSkipVerify: true,
	//})
	//
	//if err != nil {
	//}
	filename := r.genFilename(r.cfg.MetricUrl, r.cfg.JobName)
	file := filepath.Join(r.cfg.PrometheusConfDir, filename)
	err = ioutil.WriteFile(file, jBytes, os.ModePerm)
	if err != nil {
		return fmt.Errorf("写入配置文件失败:%w", err)
	}

	// 重启 prometheus
	reloadUrl := fmt.Sprintf("http://127.0.0.1:%d/-/reload", r.cfg.PrometheusPort)

	_, err = grequests.Post(reloadUrl, &grequests.RequestOptions{})

	if err != nil {
		return fmt.Errorf("重启 prometheus 失败")
	}

	logger.Successf("注册成功：\n%s", string(jBytes))

	return nil
}

func (r *registerExporter) genFilename(metricUrl string, jobType string) string {
	jobs := strings.Split(jobType, "_")
	s := fmt.Sprintf("-%s-%s-", jobType, metricUrl)
	h := md5.New()
	h.Write([]byte(s))
	m := hex.EncodeToString(h.Sum(nil))
	var job string
	if len(jobs) == 2 {
		job = jobs[0]
	}
	return fmt.Sprintf("%s.%s.json", job, m)
}

func (r *registerExporter) RunDeregister() error {
	err := r.cfg.Validator()
	if err != nil {
		return err
	}

	filename := r.genFilename(r.cfg.MetricUrl, r.cfg.JobName)
	file := filepath.Join(r.cfg.PrometheusConfDir, filename)

	if !utils.IsExists(file) {
		logger.Infof("未找到该服务，无需注销\n")
		return nil
	}

	err = os.Remove(file)
	if err != nil {
		return fmt.Errorf("删除配置文件失败:%w", err)
	}

	// 重启 prometheus
	reloadUrl := fmt.Sprintf("http://127.0.0.1:%d/-/reload", r.cfg.PrometheusPort)

	_, err = grequests.Post(reloadUrl, &grequests.RequestOptions{})

	if err != nil {
		return fmt.Errorf("重启 prometheus 失败")
	}

	logger.Successf("注销成功：%s\n", r.cfg.MetricUrl)

	return nil

}
