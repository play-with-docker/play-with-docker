package pwd

import (
	"io"
	"net"
	"time"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/provisioner"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/xid"
	"github.com/stretchr/testify/mock"
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
	dockerFactory      docker.FactoryApi
	event              event.EventApi
	storage            storage.StorageApi
	generator          IdGenerator
	clientCount        int32
	windowsProvisioner provisioner.ProvisionerApi
	dindProvisioner    provisioner.ProvisionerApi
}

type IdGenerator interface {
	NewId() string
}

type xidGenerator struct {
}

func (x xidGenerator) NewId() string {
	return xid.New().String()
}

type mockGenerator struct {
	mock.Mock
}

func (m *mockGenerator) NewId() string {
	args := m.Called()
	return args.String(0)
}

type PWDApi interface {
	SessionNew(duration time.Duration, stack string, stackName, imageName string) (*types.Session, error)
	SessionClose(session *types.Session) error
	SessionGetSmallestViewPort(session *types.Session) types.ViewPort
	SessionDeployStack(session *types.Session) error
	SessionGet(id string) *types.Session
	SessionSetup(session *types.Session, conf SessionSetupConf) error

	InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error)
	InstanceResizeTerminal(instance *types.Instance, cols, rows uint) error
	InstanceGetTerminal(instance *types.Instance) (net.Conn, error)
	InstanceUploadFromUrl(instance *types.Instance, fileName, dest, url string) error
	InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error
	InstanceGet(session *types.Session, name string) *types.Instance
	InstanceFindByIP(sessionId, ip string) *types.Instance
	InstanceDelete(session *types.Session, instance *types.Instance) error
	InstanceExec(instance *types.Instance, cmd []string) (int, error)
	InstanceAllowedImages() []string

	ClientNew(id string, session *types.Session) *types.Client
	ClientResizeViewPort(client *types.Client, cols, rows uint)
	ClientClose(client *types.Client)
	ClientCount() int
}

func NewPWD(f docker.FactoryApi, e event.EventApi, s storage.StorageApi) *pwd {
	return &pwd{dockerFactory: f, event: e, storage: s, generator: xidGenerator{}, windowsProvisioner: provisioner.NewWindows(f), dindProvisioner: provisioner.NewDinD(f)}
}

func (p *pwd) getProvisioner(t string) (provisioner.ProvisionerApi, error) {
	if t == "windows" {
		return p.windowsProvisioner, nil
	} else {
		return p.dindProvisioner, nil
	}
}

func (p *pwd) docker(sessionId string) docker.DockerApi {
	d, err := p.dockerFactory.GetForSession(sessionId)
	if err != nil {
		panic("Should not have got here. Session always need to be validated before calling this.")
	}
	return d
}

func (p *pwd) setGauges() {
	s, _ := p.storage.SessionCount()
	ses := float64(s)
	i, _ := p.storage.InstanceCount()
	ins := float64(i)
	c := p.ClientCount()
	cli := float64(c)

	clientsGauge.Set(cli)
	instancesGauge.Set(ins)
	sessionsGauge.Set(ses)
}
