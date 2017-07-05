package pwd

import (
	"io"
	"sync"
	"time"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	sessionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sessions",
		Help: "Sessions",
	})
	clientsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "clients",
		Help: "Clients",
	})
	instancesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "instances",
		Help: "Instances",
	})

	latencyHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pwd_action_duration_ms",
		Help:    "How long it took to process a specific action, in a specific host",
		Buckets: []float64{300, 1200, 5000},
	}, []string{"action"})
)

func observeAction(action string, start time.Time) {
	latencyHistogramVec.WithLabelValues(action).Observe(float64(time.Since(start).Nanoseconds()) / 1000000)
}

var sessions map[string]*Session
var sessionsMutex sync.Mutex

func init() {
	prometheus.MustRegister(sessionsGauge)
	prometheus.MustRegister(clientsGauge)
	prometheus.MustRegister(instancesGauge)
	prometheus.MustRegister(latencyHistogramVec)

	sessions = make(map[string]*Session)
}

type pwd struct {
	docker    docker.DockerApi
	tasks     SchedulerApi
	broadcast BroadcastApi
	storage   StorageApi
}

type PWDApi interface {
	SessionNew(duration time.Duration, stack string, stackName, imageName string) (*Session, error)
	SessionClose(session *Session) error
	SessionGetSmallestViewPort(session *Session) ViewPort
	SessionDeployStack(session *Session) error
	SessionGet(id string) *Session
	SessionLoadAndPrepare() error
	SessionSetup(session *Session, conf SessionSetupConf) error

	InstanceNew(session *Session, conf InstanceConfig) (*Instance, error)
	InstanceResizeTerminal(instance *Instance, cols, rows uint) error
	InstanceAttachTerminal(instance *Instance) error
	InstanceUploadFromUrl(instance *Instance, url string) error
	InstanceUploadFromReader(instance *Instance, filename string, reader io.Reader) error
	InstanceGet(session *Session, name string) *Instance
	InstanceFindByIP(ip string) *Instance
	InstanceFindByAlias(sessionPrefix, alias string) *Instance
	InstanceFindByIPAndSession(sessionPrefix, ip string) *Instance
	InstanceDelete(session *Session, instance *Instance) error
	InstanceWriteToTerminal(instance *Instance, data string)
	InstanceAllowedImages() []string
	InstanceExec(instance *Instance, cmd []string) (int, string, error)

	ClientNew(id string, session *Session) *Client
	ClientResizeViewPort(client *Client, cols, rows uint)
	ClientClose(client *Client)
}

func NewPWD(d docker.DockerApi, t SchedulerApi, b BroadcastApi, s StorageApi) *pwd {
	p := &pwd{docker: d, tasks: t, broadcast: b, storage: s}
	return p
}

func setGauges() {
	var ins float64
	var cli float64

	for _, s := range sessions {
		ins += float64(len(s.Instances))
		cli += float64(len(s.clients))
	}

	clientsGauge.Set(cli)
	instancesGauge.Set(ins)
	sessionsGauge.Set(float64(len(sessions)))
}
