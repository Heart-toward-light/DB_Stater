/*
@Author : WuWeiJian
@Date : 2020-12-18 20:46
*/

package global

const (
	ServicePath         = "/usr/lib/systemd/system"
	PackagePath         = "../package"
	ServiceTemplatePath = "../systemd"
	Md5FileName         = "md5"
	Salt                = "qaxin"
)

const (
	DbuplibName        = "dbuplib"
	DbuplibPackageName = "dbuplib_%s_%s.tar.gz"
	DbuplibPath        = "/usr/lib64"
	LdConfigFile       = "/etc/ld.so.conf.d/dbup-x86_64.conf"
)

var MissSoLibrariesAndRepairPlanList = map[string]string{
	"libcrypto.so": "yum install openssl openssl-libs compat-openssl*",
	"libatomic.so": "yum install libatomic",
}
