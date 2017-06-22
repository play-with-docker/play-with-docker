package pwd

import (
	"io"
	"time"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
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

func init() {
	prometheus.MustRegister(sessionsGauge)
	prometheus.MustRegister(clientsGauge)
	prometheus.MustRegister(instancesGauge)
	prometheus.MustRegister(latencyHistogramVec)
}

type pwd struct {
	docker    docker.DockerApi
	tasks     SchedulerApi
	broadcast BroadcastApi
	storage   storage.StorageApi
}

type PWDApi interface {
	SessionNew(duration time.Duration, stack string, stackName, imageName string) (*types.Session, error)
	SessionClose(session *types.Session) error
	SessionGetSmallestViewPort(session *types.Session) types.ViewPort
	SessionDeployStack(session *types.Session) error
	SessionGet(id string) *types.Session
	SessionSetup(session *types.Session, conf SessionSetupConf) error

	InstanceNew(session *types.Session, conf InstanceConfig) (*types.Instance, error)
	InstanceResizeTerminal(instance *types.Instance, cols, rows uint) error
	InstanceAttachTerminal(instance *types.Instance) error
	InstanceUploadFromUrl(instance *types.Instance, url string) error
	InstanceUploadFromReader(instance *types.Instance, filename string, reader io.Reader) error
	InstanceGet(session *types.Session, name string) *types.Instance
	InstanceFindByIP(ip string) *types.Instance
	InstanceFindByAlias(sessionPrefix, alias string) *types.Instance
	InstanceFindByIPAndSession(sessionPrefix, ip string) *types.Instance
	InstanceDelete(session *types.Session, instance *types.Instance) error
	InstanceWriteToTerminal(instance *types.Instance, data string)
	InstanceAllowedImages() []string
	InstanceExec(instance *types.Instance, cmd []string) (int, error)

	ClientNew(id string, session *types.Session) *types.Client
	ClientResizeViewPort(client *types.Client, cols, rows uint)
	ClientClose(client *types.Client)
}

func NewPWD(d docker.DockerApi, t SchedulerApi, b BroadcastApi, s storage.StorageApi) *pwd {
	return &pwd{docker: d, tasks: t, broadcast: b, storage: s}
}

func (p *pwd) setGauges() {
	s, _ := p.storage.SessionCount()
	ses := float64(s)
	i, _ := p.storage.InstanceCount()
	ins := float64(i)
	c, _ := p.storage.ClientCount()
	cli := float64(c)

	clientsGauge.Set(cli)
	instancesGauge.Set(ins)
	sessionsGauge.Set(ses)
}
