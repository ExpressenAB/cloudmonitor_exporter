package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/avct/user-agent-surfer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

var (
	listenAddress     = flag.String("exporter.address", ":9143", "The address on which to expose the web interface and generated Prometheus metrics.")
	namespace         = flag.String("exporter.namespace", "cloudmonitor", "The prometheus namespace.")
	metricsEndpoint   = flag.String("metrics.endpoint", "/metrics", "Path under which to expose metrics.")
	collectorEndpoint = flag.String("collector.endpoint", "/collector", "Path under which to accept cloudmonitor data.")
	accesslog         = flag.String("collector.accesslog", "", "Log incoming collector data to specified file.")
	logErrors         = flag.Bool("collector.logerrors", false, "Log errors(5..) to stdout")
	showVersion       = flag.Bool("version", false, "Show version information")
)

type Exporter struct {
	sync.RWMutex
	httpRequestsTotal, httpDeviceRequestsTotal, httpResponseContentEncodingTotal, httpGeoRequestsTotal, httpResponseBytesTotal, httpResponseContentTypesTotal, parseErrorsTotal, originRetriesTotal *prometheus.CounterVec
	httpResponseLatency, httpOriginLatency                                                                                                                                                          *prometheus.SummaryVec
	postSizeBytesTotal                                                                                                                                                                              prometheus.Counter
	postProcessingTime, logLatency                                                                                                                                                                  prometheus.Summary
	logWriter                                                                                                                                                                                       *bufio.Writer
	logfile                                                                                                                                                                                         *os.File
	writeAccesslog, logErrors                                                                                                                                                                       bool
}

type CloudmonitorStruct struct {
	Type        string            `json:"type"`
	Format      string            `json:"format"`
	Version     string            `json:"version"`
	ID          string            `json:"id"`
	Start       string            `json:"start"`
	CPCode      string            `json:"cp"`
	Message     MessageStruct     `json:"message"`
	Request     RequestStruct     `json:"reqHdr"`
	Response    ResponseStruct    `json:"respHdr"`
	Performance PerformanceStruct `json:"netPerf"`
	Network     NetworkStruct     `json:"network"`
	Geo         GeoStruct         `json:"geo"`
}

type GeoStruct struct {
	City      string `json:"city"`
	Country   string `json:"country"`
	Latitude  string `json:"lat"`
	Longitude string `json:"long"`
	Region    string `json:"region"`
}

type NetworkStruct struct {
	ASNum       string `json:"asnum"`
	Network     string `json:"network"`
	NetworkType string `json:"networkType"`
	EdgeIP      string `json:"edgeIP"`
}

type PerformanceStruct struct {
	DownloadTime      float64 `json:"downloadTime,string"`
	OriginName        string  `json:"originName"`
	OriginIP          string  `json:"originIP"`
	OriginInitIP      string  `json:"originInitIP"`
	OriginRetry       int     `json:"originRetry,string"`
	LastMileRTT       string  `json:"lastMileRTT"`
	MidMileLatency    string  `json:"midMileLatency"`
	OriginLatency     float64 `json:"netOriginLatency,string"`
	LastMileBandwidth string  `json:"lastMileBW"`
	CacheStatus       int     `json:"cacheStatus,string"`
	FirstByte         string  `json:"firstByte"`
	LastByte          string  `json:"lastByte"`
	ASNum             string  `json:"asnum"`
	Network           string  `json:"network"`
	NetworkType       string  `json:"netType"`
	EdgeIP            string  `json:"edgeIP"`
}

type MessageStruct struct {
	Protocol        string  `json:"proto"`
	ProtocolVersion string  `json:"protoVer"`
	ClientIP        string  `json:"cliIP"`
	ReqPort         string  `json:"reqPort"`
	ReqHost         string  `json:"reqHost"`
	ReqMethod       string  `json:"reqMethod"`
	ReqPath         string  `json:"reqPath"`
	ReqQuery        string  `json:"reqQuery"`
	ReqContentType  string  `json:"reqCT"`
	ReqLength       float64 `json:"reqLen"`
	SSLVer          string  `json:"sslVer"`
	ResStatus       string  `json:"status"`
	ResLocation     string  `json:"redirURL"`
	ResContentType  string  `json:"respCT"`
	ResLength       float64 `json:"respLen,string"`
	ResBytes        float64 `json:"bytes,string"`
	UserAgent       string  `json:"UA"`
	ForwardHost     string  `json:"fwdHost"`
}

type RequestStruct struct {
	AcceptEncoding    string `json:"accEnc"`
	AcceptLanguage    string `json:"accLang"`
	Authorization     string `json:"auth"`
	CacheControl      string `json:"cacheCtl"`
	Connection        string `json:"conn"`
	MD5               string `json:"contMD5"`
	Cookie            string `json:"cookie"`
	DNT               string `json:"DNT"`
	Expect            string `json:"expect"`
	IfMatch           string `json:"ifMatch"`
	IfModifiedSince   string `json:"ifMod"`
	IfNoneMatch       string `json:"ifNone"`
	IfRange           string `json:"ifRange"`
	IfUnmodifiedSince string `json:"ifUnmod"`
	Range             string `json:"range"`
	Referer           string `json:"referer"`
	TE                string `json:"te"`
	Upgrade           string `json:"upgrade"`
	Via               string `json:"via"`
	XForwardedFor     string `json:"xFrwdFor"`
	XRequestedWith    string `json:"xReqWith"`
}

type ResponseStruct struct {
	AcceptRanges             string `json:"accRange"`
	AccessControlAllowOrigin string `json:"allowOrigin"`
	Age                      string `json:"age"`
	Allow                    string `json:"allow"`
	CacheControl             string `json:"cacheCtl"`
	Connection               string `json:"conn"`
	ContentEncoding          string `json:"contEnc"`
	ContentLanguage          string `json:"contLang"`
	ContentMD5               string `json:"contMD5"`
	ContentDisposition       string `json:"contDisp"`
	ContentRange             string `json:"contRange"`
	Date                     string `json:"date"`
	ETag                     string `json:"eTag"`
	Expires                  string `json:"expires"`
	LastModified             string `json:"lastMod"`
	Link                     string `json:"link"`
	P3P                      string `json:"p3p"`
	RetryAfter               string `json:"retry"`
	Server                   string `json:"server"`
	Trailer                  string `json:"trailer"`
	TransferEncoding         string `json:"transEnc"`
	Vary                     string `json:"vary"`
	Via                      string `json:"via"`
	Warning                  string `json:"warning"`
	WWWAuthenticate          string `json:"wwwAuth"`
	XPoweredBy               string `json:"xPwrdBy"`
	SetCookie                string `json:"setCookie"`
}

func NewExporter(errors bool) *Exporter {
	return &Exporter{
		writeAccesslog: false,
		logErrors:      errors,
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_requests_total",
				Help:      "Total number of processed requests",
			},
			[]string{"host", "method", "status_code", "cache", "protocol", "protocol_version", "ip_version"},
		),
		httpDeviceRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_device_requests_total",
				Help:      "Total number of processed requests by devices",
			},
			[]string{"host", "device", "cache", "protocol", "protocol_version", "ip_version"},
		),
		httpGeoRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_geo_requests_total",
				Help:      "Total number of processed requests by country",
			},
			[]string{"host", "country", "ip_version"},
		),
		httpResponseBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_response_bytes_total",
				Help:      "Total response size in bytes",
			},
			[]string{"host", "method", "status_code", "cache", "protocol", "protocol_version"},
		),
		httpResponseContentEncodingTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_response_content_encoding_total",
				Help:      "Counter of response content encodig",
			},
			[]string{"host", "encoding", "content_type"},
		),
		httpResponseContentTypesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_response_content_types_total",
				Help:      "Counter of response content types",
			},
			[]string{"host", "cache", "content_type"},
		),
		httpResponseLatency: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: *namespace,
				Name:      "http_response_latency_milliseconds",
				Help:      "Response latency in milliseconds",
			},
			[]string{"host", "cache", "protocol", "protocol_version", "ip_version"},
		),
		httpOriginLatency: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: *namespace,
				Name:      "http_origin_latency_milliseconds",
				Help:      "Origin latency in milliseconds",
			},
			[]string{"host", "cache", "protocol", "protocol_version", "ip_version"},
		),
		logLatency: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Namespace: *namespace,
				Name:      "log_latency_seconds",
				Help:      "Summary of latency of incoming logs",
			},
		),
		originRetriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "origin_retries_total",
				Help:      "Number of origin retries",
			},
			[]string{"host", "status_code", "protocol", "ip_version"},
		),
		parseErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "parse_errors_total",
				Help:      "Number of detected parse errors",
			},
			[]string{"error"},
		),
		postProcessingTime: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Namespace: *namespace,
				Name:      "post_processing_time_seconds",
				Help:      "Seconds to process post",
			},
		),
		postSizeBytesTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "post_size_bytes_total",
				Help:      "Size of incoming postdata in bytes",
			},
		),
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.httpRequestsTotal.Collect(ch)
	e.httpDeviceRequestsTotal.Collect(ch)
	e.httpResponseContentEncodingTotal.Collect(ch)
	e.httpGeoRequestsTotal.Collect(ch)
	e.httpResponseBytesTotal.Collect(ch)
	e.httpResponseContentTypesTotal.Collect(ch)
	e.originRetriesTotal.Collect(ch)
	e.parseErrorsTotal.Collect(ch)
	e.httpResponseLatency.Collect(ch)
	e.httpOriginLatency.Collect(ch)

	ch <- e.postProcessingTime
	ch <- e.logLatency
	ch <- e.postSizeBytesTotal
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.httpRequestsTotal.Describe(ch)
	e.httpDeviceRequestsTotal.Describe(ch)
	e.httpResponseContentEncodingTotal.Describe(ch)
	e.httpGeoRequestsTotal.Describe(ch)
	e.httpResponseBytesTotal.Describe(ch)
	e.httpResponseContentTypesTotal.Describe(ch)
	e.originRetriesTotal.Describe(ch)
	e.parseErrorsTotal.Describe(ch)
	e.httpResponseLatency.Describe(ch)
	e.httpOriginLatency.Describe(ch)

	ch <- e.postProcessingTime.Desc()
	ch <- e.logLatency.Desc()
	ch <- e.postSizeBytesTotal.Desc()
}

func (e *Exporter) UnescapeString(s string) string {
	o, err := url.QueryUnescape(s)

	if err != nil {
		return ""
	}

	return o
}

func (e *Exporter) GetCacheString(i int) string {
	switch i {
	case 0:
		return "notcachable"
	case 1, 2:
		return "hit"
	case 3:
		return "miss"
	}

	return "-"
}

func (e *Exporter) Close() error {
	return e.logfile.Close()
}

func (e *Exporter) SetLogfile(logpath string) {
	if len(logpath) <= 0 {
		return
	}

	var err error

	e.logfile, err = os.OpenFile(logpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	e.logWriter = bufio.NewWriter(e.logfile)
	e.writeAccesslog = true
}

func (e *Exporter) OutputLogEntry(cloudmonitorData *CloudmonitorStruct) {
	query := ""

	if len(cloudmonitorData.Message.ReqQuery) > 0 {
		query = "?" + cloudmonitorData.Message.ReqQuery
	}

	logentry := fmt.Sprintf("%s %s %s \"%s %s://%s%s%s %s HTTP/%s\" %s %v '%s'\n",
		cloudmonitorData.Message.ClientIP,
		cloudmonitorData.Network.EdgeIP,
		e.MillisecondsToTime(cloudmonitorData.Start),
		cloudmonitorData.Message.ReqMethod,
		cloudmonitorData.Message.Protocol,
		cloudmonitorData.Message.ReqHost,
		e.UnescapeString(cloudmonitorData.Message.ReqPath),
		e.UnescapeString(query),
		cloudmonitorData.Message.ResStatus,
		cloudmonitorData.Message.ProtocolVersion,
		e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
		cloudmonitorData.Message.ResBytes,
		cloudmonitorData.Message.UserAgent)

	if e.writeAccesslog == true {
		fmt.Fprintf(e.logWriter, logentry)
	}

	if e.logErrors {
		status, _ := strconv.Atoi(cloudmonitorData.Message.ResStatus)
		if status >= 500 && status <= 599 {
			fmt.Printf(logentry)
		}
	}
}

func (e *Exporter) DummyUse(vals ...interface{}) {
	for _, val := range vals {
		_ = val
	}
}

func (e *Exporter) GetDeviceType(userAgent string) string {

	ua := uasurfer.Parse(userAgent)

	switch ua.DeviceType {
	case uasurfer.DeviceComputer:
		return "desktop"
	case uasurfer.DevicePhone:
		return "mobile"
	case uasurfer.DeviceTablet:
		return "tablet"
	case uasurfer.DeviceTV:
		return "tv"
	case uasurfer.DeviceConsole:
		return "console"
	case uasurfer.DeviceWearable:
		return "wearable"
	default:
		return "unknown"
	}
}

func (e *Exporter) MillisecondsToTime(ms string) time.Time {
	i, err := strconv.ParseFloat(ms, 64)
	if err != nil {
		return time.Now()
	}
	return time.Unix(int64(i), 0)
}

func (e *Exporter) ReportParseError(error string) {
	e.parseErrorsTotal.WithLabelValues(error).Inc()
}

func getIPVersion(ip_s string) string {
	ip := net.ParseIP(ip_s)
	if ip.To4() != nil {
		return "ipv4"
	} else if ip.To16() != nil {
		return "ipv6"
	} else {
		return "unknown"
	}
}

func (e *Exporter) HandleCollectorPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Internal server error", http.StatusMethodNotAllowed)
		return
	}

	var multiplier float64 = float64(1)
	dir, file := path.Split(r.URL.Path)
	if dir != "" && path.Base(dir) == "sample-percentage" {
		if sample, err := strconv.Atoi(file); err != nil || sample == 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else {
			multiplier = float64(100 / sample)
		}
	}

	e.postSizeBytesTotal.Add(float64(r.ContentLength))

	begin := time.Now()

	scanner := bufio.NewScanner(r.Body)
	defer r.Body.Close()

	for scanner.Scan() {

		cloudmonitorData := &CloudmonitorStruct{}

		if err := json.NewDecoder(strings.NewReader(scanner.Text())).Decode(cloudmonitorData); err != nil {
			e.ReportParseError(err.Error())
			log.Printf("Could not parse message %q (%v)\n", scanner.Text(), err)
			continue
		}

		ipVersion := getIPVersion(cloudmonitorData.Message.ClientIP)

		e.OutputLogEntry(cloudmonitorData)

		e.httpRequestsTotal.WithLabelValues(
			cloudmonitorData.Message.ReqHost,
			cloudmonitorData.Message.ReqMethod,
			string(cloudmonitorData.Message.ResStatus),
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
			cloudmonitorData.Message.Protocol,
			cloudmonitorData.Message.ProtocolVersion,
			ipVersion,
		).Add(multiplier)

		deviceType := e.GetDeviceType(e.UnescapeString(cloudmonitorData.Message.UserAgent))

		e.httpDeviceRequestsTotal.WithLabelValues(
			cloudmonitorData.Message.ReqHost,
			deviceType,
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
			cloudmonitorData.Message.Protocol,
			cloudmonitorData.Message.ProtocolVersion,
			ipVersion,
		).Add(multiplier)

		// Don't increment for non-defined content-types
		if cloudmonitorData.Message.ResContentType != "" && cloudmonitorData.Message.ResContentType != "content_type" {
			e.httpResponseContentEncodingTotal.WithLabelValues(
				cloudmonitorData.Message.ReqHost,
				strings.ToLower(string(cloudmonitorData.Response.ContentEncoding)),
				strings.ToLower(string(cloudmonitorData.Message.ResContentType)),
			).Add(multiplier)

			e.httpResponseContentTypesTotal.WithLabelValues(
				cloudmonitorData.Message.ReqHost,
				e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
				strings.ToLower(string(cloudmonitorData.Message.ResContentType)),
			).Add(multiplier)
		}

		e.httpResponseBytesTotal.WithLabelValues(
			cloudmonitorData.Message.ReqHost,
			cloudmonitorData.Message.ReqMethod,
			string(cloudmonitorData.Message.ResStatus),
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
			cloudmonitorData.Message.Protocol,
			cloudmonitorData.Message.ProtocolVersion,
		).Add(cloudmonitorData.Message.ResBytes * multiplier)

		e.httpGeoRequestsTotal.WithLabelValues(
			cloudmonitorData.Message.ReqHost,
			cloudmonitorData.Geo.Country,
			ipVersion,
		).Add(multiplier)

		e.httpResponseLatency.WithLabelValues(
			cloudmonitorData.Message.ReqHost,
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
			cloudmonitorData.Message.Protocol,
			cloudmonitorData.Message.ProtocolVersion,
			ipVersion,
		).Observe(cloudmonitorData.Performance.DownloadTime)

		e.httpOriginLatency.WithLabelValues(
			cloudmonitorData.Message.ReqHost,
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
			cloudmonitorData.Message.Protocol,
			cloudmonitorData.Message.ProtocolVersion,
			ipVersion,
		).Observe(cloudmonitorData.Performance.OriginLatency)

		latency := time.Since(e.MillisecondsToTime(cloudmonitorData.Start))
		e.logLatency.Observe(latency.Seconds())

		e.originRetriesTotal.WithLabelValues(
			cloudmonitorData.Message.ReqHost,
			string(cloudmonitorData.Message.ResStatus),
			cloudmonitorData.Message.Protocol,
			ipVersion,
		).Add(float64(cloudmonitorData.Performance.OriginRetry) * multiplier)
	}

	duration := time.Since(begin)
	e.postProcessingTime.Observe(duration.Seconds())

	if e.writeAccesslog {
		e.logWriter.Flush()
	}

}

func main() {

	flag.Parse()

	log.Printf("Cloudmonitor-exporter %s\n", version.Print("cloudmonitor_exporter"))
	if *showVersion {
		return
	}

	exporter := NewExporter(*logErrors)
	defer exporter.Close()

	if len(*accesslog) > 0 {
		exporter.SetLogfile(*accesslog)
		log.Printf("logging to %s", *accesslog)
	}

	prometheus.MustRegister(version.NewCollector("cloudmonitor_exporter"))
	prometheus.MustRegister(exporter)

	if !strings.HasSuffix(*collectorEndpoint, "/") {
		endpointWithSlash := fmt.Sprintf("%v/", *collectorEndpoint)
		http.HandleFunc(endpointWithSlash, exporter.HandleCollectorPost)
	}

	http.Handle(*metricsEndpoint, prometheus.Handler())
	http.HandleFunc(*collectorEndpoint, exporter.HandleCollectorPost)

	log.Printf("providing metrics at %s%s", *listenAddress, *metricsEndpoint)
	log.Printf("accepting logs at at %s%s", *listenAddress, *collectorEndpoint)

	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
