package influxdb_output

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/karimra/gnmic/collector"
	"github.com/karimra/gnmic/outputs"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/proto"
)

const (
	defaultURL               = "http://localhost:8086"
	defaultBatchSize         = 1000
	defaultFlushTimer        = 10 * time.Second
	defaultHealthCheckPeriod = 30 * time.Second

	numWorkers = 1
)

func init() {
	outputs.Register("influxdb", func() outputs.Output {
		return &InfluxDBOutput{
			Cfg:       &Config{},
			eventChan: make(chan *collector.EventMsg),
			reset:     make(chan struct{}),
			startSig:  make(chan struct{}),
		}
	})
}

type InfluxDBOutput struct {
	Cfg       *Config
	client    influxdb2.Client
	metrics   []prometheus.Collector
	logger    *log.Logger
	cancelFn  context.CancelFunc
	eventChan chan *collector.EventMsg
	reset     chan struct{}
	startSig  chan struct{}
	wasup     bool
}
type Config struct {
	URL               string        `mapstructure:"url,omitempty"`
	Org               string        `mapstructure:"org,omitempty"`
	Bucket            string        `mapstructure:"bucket,omitempty"`
	Token             string        `mapstructure:"token,omitempty"`
	BatchSize         uint          `mapstructure:"batch_size,omitempty"`
	FlushTimer        time.Duration `mapstructure:"flush_timer,omitempty"`
	UseGzip           bool          `mapstructure:"use_gzip,omitempty"`
	EnableTLS         bool          `mapstructure:"enable_tls,omitempty"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period,omitempty"`
	Debug             bool          `mapstructure:"debug,omitempty"`
}

func (k *InfluxDBOutput) String() string {
	b, err := json.Marshal(k)
	if err != nil {
		return ""
	}
	return string(b)
}
func (i *InfluxDBOutput) SetLogger(logger *log.Logger) {
	if logger != nil {
		i.logger = log.New(logger.Writer(), "influxdb_output ", logger.Flags())
		return
	}
	i.logger = log.New(os.Stderr, "influxdb_output ", log.LstdFlags|log.Lmicroseconds)
}

func (i *InfluxDBOutput) Init(ctx context.Context, cfg map[string]interface{}, opts ...outputs.Option) error {
	for _, opt := range opts {
		opt(i)
	}
	err := outputs.DecodeConfig(cfg, i.Cfg)
	if err != nil {
		i.logger.Printf("influxdb output config decode failed: %v", err)
		return err
	}
	if i.Cfg.URL == "" {
		i.Cfg.URL = defaultURL
	}
	if i.Cfg.BatchSize == 0 {
		i.Cfg.BatchSize = defaultBatchSize
	}
	if i.Cfg.FlushTimer == 0 {
		i.Cfg.FlushTimer = defaultFlushTimer
	}
	if i.Cfg.HealthCheckPeriod == 0 {
		i.Cfg.HealthCheckPeriod = defaultHealthCheckPeriod
	}

	iopts := influxdb2.DefaultOptions().
		SetUseGZip(i.Cfg.UseGzip).
		SetBatchSize(i.Cfg.BatchSize).
		SetFlushInterval(uint(i.Cfg.FlushTimer.Milliseconds()))
	if i.Cfg.EnableTLS {
		iopts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if i.Cfg.Debug {
		iopts.SetLogLevel(3)
	}
	ctx, i.cancelFn = context.WithCancel(ctx)
CRCLIENT:
	i.client = influxdb2.NewClientWithOptions(i.Cfg.URL, i.Cfg.Token, iopts)
	// start influx health check
	err = i.health(ctx)
	if err != nil {
		log.Printf("failed to check influxdb health: %v", err)
		time.Sleep(10 * time.Second)
		goto CRCLIENT
	}
	i.wasup = true
	go i.healthCheck(ctx)
	i.logger.Printf("initialized influxdb client: %s", i.String())

	for k := 0; k < numWorkers; k++ {
		go i.worker(ctx, k)
	}
	go func() {
		<-ctx.Done()
		i.Close()
	}()
	return nil
}

func (i *InfluxDBOutput) Write(ctx context.Context, rsp proto.Message, meta outputs.Meta) {
	if rsp == nil {
		return
	}
	switch rsp := rsp.(type) {
	case *gnmi.SubscribeResponse:
		measName := "default"
		if subName, ok := meta["subscription-name"]; ok {
			measName = subName
		}
		events, err := collector.ResponseToEventMsgs(measName, rsp, meta)
		if err != nil {
			i.logger.Printf("failed to convert message to event: %v", err)
			return
		}
		for _, ev := range events {
			select {
			case <-ctx.Done():
				return
			case <-i.reset:
				return
			case i.eventChan <- ev:
			}
		}
	}
}

func (i *InfluxDBOutput) Close() error {
	i.logger.Printf("closing client...")
	i.cancelFn()
	i.logger.Printf("closed.")
	return nil
}
func (i *InfluxDBOutput) Metrics() []prometheus.Collector { return i.metrics }

func (i *InfluxDBOutput) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(i.Cfg.HealthCheckPeriod)
	for {
		select {
		case <-ticker.C:
			i.health(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (i *InfluxDBOutput) health(ctx context.Context) error {
	res, err := i.client.Health(ctx)
	if err != nil {
		i.logger.Printf("failed health check: %v", err)
		if i.wasup {
			close(i.reset)
			i.reset = make(chan struct{})
		}
		return err
	}
	if res != nil {
		b, err := json.Marshal(res)
		if err != nil {
			i.logger.Printf("failed to marshal health check result: %v", err)
			i.logger.Printf("health check result: %+v", res)
			if i.wasup {
				close(i.reset)
				i.reset = make(chan struct{})
			}
			return err
		}
		i.wasup = true
		close(i.startSig)
		i.startSig = make(chan struct{})
		i.logger.Printf("health check result: %s", string(b))
		return nil
	}
	i.wasup = true
	close(i.startSig)
	i.startSig = make(chan struct{})
	i.logger.Print("health check result is nil")
	return nil
}

func (i *InfluxDBOutput) worker(ctx context.Context, idx int) {
	firstStart := true
START:
	if !firstStart {
		i.logger.Printf("worker-%d waiting for client recovery", idx)
		<-i.startSig
	}
	i.logger.Printf("starting worker-%d", idx)
	writer := i.client.WriteAPI(i.Cfg.Org, i.Cfg.Bucket)
	//defer writer.Flush()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				i.logger.Printf("worker-%d err=%v", idx, ctx.Err())
			}
			i.logger.Printf("worker-%d terminating...", idx)
			return
		case ev := <-i.eventChan:
			writer.WritePoint(influxdb2.NewPoint(ev.Name, ev.Tags, ev.Values, time.Unix(0, ev.Timestamp)))
		case <-i.reset:
			firstStart = false
			i.logger.Printf("resetting worker-%d...", idx)
			goto START
		case err := <-writer.Errors():
			i.logger.Printf("worker-%d write error: %v", idx, err)
		}
	}
}
