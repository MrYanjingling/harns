package util

const (
	GatewayTCPPort = 80

	APT = "apt"
	YUM = "yum"

	TOOLMOSQUITTO          = "mosquitto"
	TOOLEDGIOT             = "edgeiot"
	TOOLINFLUXDB           = "influxdb"
	TOOLINFLUXDBSERVEREXEC = "influxd"

	MemInfoConfigPath      = "/proc/meminfo/"
	MemTotalField          = "MemTotal"
	MemMinSize             = 15 * 1024 * 1024

	OneHour        = 1
	OneDayInHours  = 24 * OneHour
	OneWeekInHours = 7 * OneDayInHours
)

// NginxConfTemplate https://www.slashroot.in/nginx-web-server-performance-tuning-how-to-do-it
// https://segmentfault.com/a/1190000016385662
// http://nginx.org/en/docs/ngx_core_module.html
// https://blog.martinfjordvald.com/optimizing-nginx-for-high-traffic-loads/?utm_source=rss&utm_medium=rss&utm_campaign=optimizing-nginx-for-high-traffic-loads
