package constant

var SysctlConfPath = "/proc/sys/net/ipv4/conf"

// var disableIPv6SysctlTemplate = "net/ipv6/conf/%s/disable_ipv6"

var DefaultInterfacesToExclude = []string{
	"docker.*", "cbr.*", "dummy.*",
	"virbr.*", "lxcbr.*", "veth.*", "lo",
	"cali.*", "tunl.*", "flannel.*", "kube-ipvs.*", "cni.*",
}

var OverlayRouteTable = 100
var DefaultInterfaceName = "eth0"
var DefaultMacPrefix = "80:80"

var ErrRouteFileExist string = "file exists"

// Log level character string
const (
	LogDebugLevelStr = "debug"
	LogInfoLevelStr  = "info"
	LogWarnLevelStr  = "warn"
	LogErrorLevelStr = "error"
	LogFatalLevelStr = "fatal"
	LogPanicLevelStr = "panic"
)

const (
	VethLogDefaultFilePath   = "/var/log/meta-plugins/veth.log"
	RouterLogDefaultFilePath = "/var/log/meta-plugins/router.log"
	LogDefaultMaxSize        = 100 // megabytes
	LogDefaultMaxAge         = 5   // days
	LogDefaultMaxBackups     = 5
)
