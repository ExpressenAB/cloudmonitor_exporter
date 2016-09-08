package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	listenAddress     = flag.String("exporter.address", ":9143", "The address on which to expose the web interface and generated Prometheus metrics.")
	namespace         = flag.String("exporter.namespace", "cloudmonitor", "The prometheus namespace.")
	metricsEndpoint   = flag.String("metrics.endpoint", "/metrics", "Path under which to expose metrics.")
	collectorEndpoint = flag.String("collector.endpoint", "/collector", "Path under which to accept cloudmonitor data.")
	accesslog         = flag.String("collector.accesslog", "", "Log incoming collector data to specified file.")
	showVersion       = flag.Bool("version", false, "Show version information")
	version           = "0.1.1"
)

type Exporter struct {
	sync.RWMutex
	startTime                                                                                           time.Time
	httpRequestsTotal, httpResponseSizeBytes, httpResponseContentTypes, parseErrors, originRetriesTotal *prometheus.CounterVec
	httpResponseLatency, httpOriginLatency                                                                 *prometheus.SummaryVec
	exporterUptime, postSize                                                                            prometheus.Counter
	postProcessingTime, logLatency                                                                      prometheus.Summary
	logWriter                                                                                           *bufio.Writer
	writeAccesslog                                                                                      bool
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
	Response    ResponseStruct    `json:"resHdr"`
	Performance PerformanceStruct `json:"netPerf"`
	Network     NetworkStruct     `json:"network"`
}

type NetworkStruct struct {
	ASNum       string `json:"asnum"`
	Network     string `json:"network"`
	NetworkType string `json:"networkType"`
	EdgeIP      string `json:"edgeIP`
}

type PerformanceStruct struct {
	DownloadTime      float64 `json:"downloadTime,string"`
	OriginName        string  `json:"originName"`
	OriginIP          string  `json:"originIP"`
	OriginInitIP      string  `json:"originInitIP"`
	OriginRetry       int     `json:"originRetry,string"`
	LastMileRTT       string  `json:"lastMileRTT"`
	MidMileLatency    string  `json:"midMileLatency"`
	OriginLatency     float64 `json:"netOriginLatency"`
	LastMileBandwidth string  `json:"lastMileBW"`
	CacheStatus       int     `json:"cacheStatus,string"`
	FirstByte         string  `json:"firstByte"`
	LastByte          string  `json:"lastByte"`
	ASNum             string  `json:"asnum"`
	Network           string  `json:"network"`
	NetworkType       string  `json:"netType"`
	EdgeIP            string  `json:"edgeIP`
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
	ResBytes        string  `json:"bytes"`
	UserAgent       string  `json:"UA"`
	ForwardHost     string  `json:"fwdHost`
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

func NewExporter() *Exporter {
	return &Exporter{
		startTime:      time.Now(),
		writeAccesslog: false,
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_requests_total",
				Help:      "Total number of processed loglines",
			},
			[]string{"host", "method", "status_code", "cache", "protocol"},
		),
		httpResponseSizeBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_response_size_bytes",
				Help:      "Total response size in bytes",
			},
			[]string{"host", "method", "status_code", "cache", "protocol"},
		),
		httpResponseContentTypes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "http_response_content_types",
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
			[]string{"host", "cache"},
		),
		httpOriginLatency: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: *namespace,
				Name:      "http_origin_latency_milliseconds",
				Help:      "Origin latency in milliseconds",
			},
			[]string{"host", "cache"},
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
			[]string{"host", "status_code", "protocol"},
		),
		exporterUptime: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "exporter_uptime_seconds",
				Help:      "Uptime of exporter",
			},
		),
		parseErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "parse_errors_count",
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
		postSize: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: *namespace,
				Name:      "post_size_bytes",
				Help:      "Size of incoming postdata in bytes",
			},
		),
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.exporterUptime.Set(time.Since(e.startTime).Seconds())

	e.httpRequestsTotal.Collect(ch)
	e.httpResponseSizeBytes.Collect(ch)
	e.httpResponseContentTypes.Collect(ch)
	e.originRetriesTotal.Collect(ch)
	e.parseErrors.Collect(ch)
	e.httpResponseLatency.Collect(ch)
	e.httpOriginLatency.Collect(ch)

	ch <- e.exporterUptime
	ch <- e.postProcessingTime
	ch <- e.logLatency
	ch <- e.postSize
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.httpRequestsTotal.Describe(ch)
	e.httpResponseSizeBytes.Describe(ch)
	e.httpResponseContentTypes.Describe(ch)
	e.originRetriesTotal.Describe(ch)
	e.parseErrors.Describe(ch)
	e.httpResponseLatency.Describe(ch)
	e.httpOriginLatency.Describe(ch)

	ch <- e.exporterUptime.Desc()
	ch <- e.postProcessingTime.Desc()
	ch <- e.logLatency.Desc()
	ch <- e.postSize.Desc()
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

func (e *Exporter) SetLogfile(logpath string) {
	if len(logpath) > 0 {
		logfile, err := os.OpenFile(logpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		e.logWriter = bufio.NewWriter(logfile)
		e.writeAccesslog = true
	}
}

func (e *Exporter) OutputLogEntry(cloudmonitorData *CloudmonitorStruct) {
	query := ""

	if len(cloudmonitorData.Message.ReqQuery) > 0 {
		query = "?" + cloudmonitorData.Message.ReqQuery
	}

	logentry := fmt.Sprintf("%s %s %s \"%s %s://%s%s%s %s HTTP/%s\" %s %v\n",
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
		cloudmonitorData.Message.ResLength)

	if e.writeAccesslog == true {
		fmt.Fprintf(e.logWriter, logentry)
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
	e.parseErrors.WithLabelValues(error).Inc()
}

func (e *Exporter) HandleCollectorPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	e.postSize.Add(float64(r.ContentLength))

	begin := time.Now()
	body := bufio.NewReader(r.Body)

	for {
		data, err := body.ReadString('\n')

		if err == io.EOF {
			break
		} else if err != nil {
			e.ReportParseError(err.Error())
			continue
		}

		cloudmonitorData := &CloudmonitorStruct{}

		if err := json.NewDecoder(strings.NewReader(data)).Decode(cloudmonitorData); err != nil {
			e.ReportParseError(err.Error())
			continue
		}

		e.OutputLogEntry(cloudmonitorData)

		e.httpRequestsTotal.WithLabelValues(cloudmonitorData.Message.ReqHost,
			cloudmonitorData.Message.ReqMethod,
			string(cloudmonitorData.Message.ResStatus),
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
			cloudmonitorData.Message.Protocol).
			Inc()

		e.httpResponseSizeBytes.WithLabelValues(cloudmonitorData.Message.ReqHost,
			cloudmonitorData.Message.ReqMethod,
			string(cloudmonitorData.Message.ResStatus),
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
			cloudmonitorData.Message.Protocol).
			Add(cloudmonitorData.Message.ResLength)

		e.httpResponseContentTypes.WithLabelValues(cloudmonitorData.Message.ReqHost,
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus),
			strings.ToLower(string(cloudmonitorData.Message.ResContentType))).
			Inc()

		e.httpResponseLatency.WithLabelValues(cloudmonitorData.Message.ReqHost,
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus)).
			Observe(cloudmonitorData.Performance.DownloadTime)

		e.httpOriginLatency.WithLabelValues(cloudmonitorData.Message.ReqHost,
			e.GetCacheString(cloudmonitorData.Performance.CacheStatus)).
			Observe(cloudmonitorData.Performance.OriginLatency)

		latency := time.Since(e.MillisecondsToTime(cloudmonitorData.Start))
		e.logLatency.Observe(latency.Seconds())

		e.originRetriesTotal.WithLabelValues(cloudmonitorData.Message.ReqHost,
			string(cloudmonitorData.Message.ResStatus),
			cloudmonitorData.Message.Protocol).
			Add(float64(cloudmonitorData.Performance.OriginRetry))
	}

	duration := time.Since(begin)
	e.postProcessingTime.Observe(duration.Seconds())

	if e.writeAccesslog {
		e.logWriter.Flush()
	}

}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("Cloudmonitor-exporter v%s\n", version)
		return
	}

	exporter := NewExporter()

	if len(*accesslog) > 0 {
		exporter.SetLogfile(*accesslog)
		log.Printf("logging to %s", *accesslog)
	}

	prometheus.MustRegister(exporter)

	http.Handle(*metricsEndpoint, prometheus.Handler())
	http.HandleFunc(*collectorEndpoint, exporter.HandleCollectorPost)

	log.Printf("providing metrics at %s%s", *listenAddress, *metricsEndpoint)
	log.Printf("accepting logs at at %s%s", *listenAddress, *collectorEndpoint)

	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
