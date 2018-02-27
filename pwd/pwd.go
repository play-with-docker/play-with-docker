package pwd

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/id"
	"github.com/play-with-docker/play-with-docker/provisioner"
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
	dockerFactory              docker.FactoryApi
	event                      event.EventApi
	storage                    storage.StorageApi
	generator                  id.Generator
	clientCount                int32
	sessionProvisioner         provisioner.SessionProvisionerApi
	instanceProvisionerFactory provisioner.InstanceProvisionerFactoryApi
	windowsProvisioner         provisioner.InstanceProvisionerApi
	dindProvisioner            provisioner.InstanceProvisionerApi
}

var sessionNotEmpty = errors.New("Session is not empty")

func SessionNotEmpty(e error) bool {
	return e == sessionNotEmpty
}

type PWDApi interface {
	SessionNew(ctx context.Context, config types.SessionConfig) (*types.Session, error)
	SessionClose(session *types.Session) error
	SessionGetSmallestViewPort(sessionId string) types.ViewPort
	SessionDeployStack(session *types.Session) error
	SessionGet(id string) (*types.Session, error)
	SessionSetup(session *types.Session, conf SessionSetupConf) error

	InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error)
	InstanceResizeTerminal(instance *types.Instance, cols, rows uint) error
	InstanceGetTerminal(instance *types.Instance) (net.Conn, error)
	InstanceUploadFromUrl(instance *types.Instance, fileName, dest, url string) error
	InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error
	InstanceGet(session *types.Session, name string) *types.Instance
	InstanceFindBySession(session *types.Session) ([]*types.Instance, error)
	InstanceDelete(session *types.Session, instance *types.Instance) error
	InstanceExec(instance *types.Instance, cmd []string) (int, error)
	InstanceFSTree(instance *types.Instance) (io.Reader, error)
	InstanceFile(instance *types.Instance, filePath string) (io.Reader, error)

	ClientNew(id string, session *types.Session) *types.Client
	ClientResizeViewPort(client *types.Client, cols, rows uint)
	ClientClose(client *types.Client)
	ClientCount() int

	UserNewLoginRequest(providerName string) (*types.LoginRequest, error)
	UserGetLoginRequest(id string) (*types.LoginRequest, error)
	UserLogin(loginRequest *types.LoginRequest, user *types.User) (*types.User, error)
	UserGet(id string) (*types.User, error)

	PlaygroundNew(playground types.Playground) (*types.Playground, error)
	PlaygroundGet(id string) *types.Playground
	PlaygroundFindByDomain(domain string) *types.Playground
	PlaygroundList() ([]*types.Playground, error)
}

func NewPWD(f docker.FactoryApi, e event.EventApi, s storage.StorageApi, sp provisioner.SessionProvisionerApi, ipf provisioner.InstanceProvisionerFactoryApi) *pwd {
	//  windowsProvisioner: provisioner.NewWindowsASG(f, s), dindProvisioner: provisioner.NewDinD(f)
	return &pwd{dockerFactory: f, event: e, storage: s, generator: id.XIDGenerator{}, sessionProvisioner: sp, instanceProvisionerFactory: ipf}
}

func (p *pwd) getProvisioner(t string) (provisioner.InstanceProvisionerApi, error) {
	return p.instanceProvisionerFactory.GetProvisioner(t)
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
