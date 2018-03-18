package pwd

import (
	"context"
	"fmt"
	"log"
	"math"
	"path"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

var preparedSessions = map[string]bool{}

type sessionBuilderWriter struct {
	sessionId string
	event     event.EventApi
}

func (s *sessionBuilderWriter) Write(p []byte) (n int, err error) {
	s.event.Emit(event.SESSION_BUILDER_OUT, s.sessionId, string(p))
	return len(p), nil
}

type SessionSetupConf struct {
	Instances      []SessionSetupInstanceConf `json:"instances"`
	PlaygroundFQDN string
}

type SessionSetupInstanceConf struct {
	Image          string     `json:"image"`
	Hostname       string     `json:"hostname"`
	IsSwarmManager bool       `json:"is_swarm_manager"`
	IsSwarmWorker  bool       `json:"is_swarm_worker"`
	Type           string     `json:"type"`
	Run            [][]string `json:"run"`
	Tls            bool       `json:"tls"`
}

func (p *pwd) SessionNew(ctx context.Context, config types.SessionConfig) (*types.Session, error) {
	defer observeAction("SessionNew", time.Now())

	s := &types.Session{}
	s.Id = p.generator.NewId()
	s.CreatedAt = time.Now()
	s.ExpiresAt = s.CreatedAt.Add(config.Duration)
	s.Ready = true
	s.Stack = config.Stack
	s.UserId = config.UserId
	s.PlaygroundId = config.Playground.Id

	if s.Stack != "" {
		s.Ready = false
	}
	stackName := config.StackName
	if stackName == "" {
		stackName = "pwd"
	}
	s.StackName = stackName
	s.ImageName = config.ImageName

	log.Printf("NewSession id=[%s]\n", s.Id)
	if err := p.sessionProvisioner.SessionNew(ctx, s); err != nil {
		log.Println(err)
		return nil, err
	}

	if err := p.storage.SessionPut(s); err != nil {
		log.Println(err)
		return nil, err
	}

	p.setGauges()
	p.event.Emit(event.SESSION_NEW, s.Id)

	return s, nil
}

func (p *pwd) SessionClose(s *types.Session) error {
	defer observeAction("SessionClose", time.Now())

	log.Printf("Starting clean up of session [%s]\n", s.Id)
	g, _ := errgroup.WithContext(context.Background())
	instances, err := p.storage.InstanceFindBySessionId(s.Id)
	if err != nil {
		log.Printf("Could not find instances in session %s. Got %v\n", s.Id, err)
		return err
	}
	for _, i := range instances {
		i := i
		g.Go(func() error {
			return p.InstanceDelete(s, i)
		})
	}
	err = g.Wait()
	if err != nil {
		log.Println(err)
		return err
	}

	if err := p.sessionProvisioner.SessionClose(s); err != nil {
		log.Println(err)
		return err
	}

	err = p.storage.SessionDelete(s.Id)
	if err != nil {
		return err
	}

	log.Printf("Cleaned up session [%s]\n", s.Id)
	p.setGauges()
	p.event.Emit(event.SESSION_END, s.Id)
	return nil

}

func (p *pwd) SessionGetSmallestViewPort(sessionId string) types.ViewPort {
	defer observeAction("SessionGetSmallestViewPort", time.Now())

	clients, err := p.storage.ClientFindBySessionId(sessionId)
	if err != nil {
		log.Printf("Error finding clients for session [%s]. Got: %v\n", sessionId, err)
		return types.ViewPort{Rows: 24, Cols: 80}
	}
	if len(clients) == 0 {
		log.Printf("Session [%s] doesn't have clients. Returning default viewport\n", sessionId)
		return types.ViewPort{Rows: 24, Cols: 80}
	}
	var minRows uint
	var minCols uint

	for _, c := range clients {
		if c.ViewPort.Rows > 0 && c.ViewPort.Cols > 0 {
			minRows = uint(c.ViewPort.Rows)
			minCols = uint(c.ViewPort.Cols)
			break
		}
	}

	for _, c := range clients {
		if c.ViewPort.Rows > 0 && c.ViewPort.Cols > 0 {
			minRows = uint(math.Min(float64(minRows), float64(c.ViewPort.Rows)))
			minCols = uint(math.Min(float64(minCols), float64(c.ViewPort.Cols)))
		}
	}

	return types.ViewPort{Rows: minRows, Cols: minCols}
}

func (p *pwd) SessionDeployStack(s *types.Session) error {
	defer observeAction("SessionDeployStack", time.Now())

	if s.Ready {
		// a stack was already deployed on this session, just ignore
		return nil
	}

	s.Ready = false
	p.event.Emit(event.SESSION_READY, s.Id, false)
	i, err := p.InstanceNew(s, types.InstanceConfig{ImageName: s.ImageName, PlaygroundFQDN: s.Host})
	if err != nil {
		log.Printf("Error creating instance for stack [%s]: %s\n", s.Stack, err)
		return err
	}

	_, fileName := filepath.Split(s.Stack)
	err = p.InstanceUploadFromUrl(i, fileName, "/var/run/pwd/uploads", s.Stack)
	if err != nil {
		log.Printf("Error uploading stack file [%s]: %s\n", s.Stack, err)
		return err
	}

	fileName = path.Base(s.Stack)
	file := fmt.Sprintf("/var/run/pwd/uploads/%s", fileName)
	cmd := fmt.Sprintf("docker swarm init --advertise-addr eth0 && docker-compose -f %s pull && docker stack deploy -c %s %s", file, file, s.StackName)

	w := sessionBuilderWriter{sessionId: s.Id, event: p.event}

	dockerClient, err := p.dockerFactory.GetForSession(s)
	if err != nil {
		log.Println(err)
		return err
	}

	code, err := dockerClient.ExecAttach(i.Name, []string{"sh", "-c", cmd}, &w)
	if err != nil {
		log.Printf("Error executing stack [%s]: %s\n", s.Stack, err)
		return err
	}

	log.Printf("Stack execution finished with code %d\n", code)
	s.Ready = true
	p.event.Emit(event.SESSION_READY, s.Id, true)
	if err := p.storage.SessionPut(s); err != nil {
		return err
	}
	return nil
}

func (p *pwd) SessionGet(sessionId string) (*types.Session, error) {
	defer observeAction("SessionGet", time.Now())

	s, err := p.storage.SessionGet(sessionId)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return s, nil
}

func (p *pwd) SessionSetup(session *types.Session, sconf SessionSetupConf) error {
	defer observeAction("SessionSetup", time.Now())

	c := sync.NewCond(&sync.Mutex{})

	var tokens *docker.SwarmTokens = nil
	var firstSwarmManager *types.Instance = nil

	instances, err := p.storage.InstanceFindBySessionId(session.Id)
	if err != nil {
		log.Println(err)
		return err
	}
	if len(instances) > 0 {
		return sessionNotEmpty
	}

	g, ctx := errgroup.WithContext(context.Background())

	for _, conf := range sconf.Instances {
		conf := conf
		g.Go(func() error {
			instanceConf := types.InstanceConfig{
				ImageName:      conf.Image,
				Hostname:       conf.Hostname,
				PlaygroundFQDN: sconf.PlaygroundFQDN,
				Type:           conf.Type,
				Tls:            conf.Tls,
			}
			i, err := p.InstanceNew(session, instanceConf)
			if err != nil {
				return err
			}

			if conf.IsSwarmManager || conf.IsSwarmWorker {
				dockerClient, err := p.dockerFactory.GetForInstance(i)
				if err != nil {
					return err
				}
				if conf.IsSwarmManager {
					c.L.Lock()
					if firstSwarmManager == nil {
						tkns, err := dockerClient.SwarmInit(i.IP)
						if err != nil {
							log.Printf("Cannot initialize swarm on instance %s. Got: %v\n", i.Name, err)
							return err
						}
						tokens = tkns
						firstSwarmManager = i
						c.Broadcast()
						c.L.Unlock()
					} else {
						c.L.Unlock()
						if err := dockerClient.SwarmJoin(fmt.Sprintf("%s:2377", firstSwarmManager.IP), tokens.Manager); err != nil {
							log.Printf("Cannot join manager %s to swarm. Got: %v\n", i.Name, err)
							return err
						}
					}
				} else if conf.IsSwarmWorker {
					c.L.Lock()
					if firstSwarmManager == nil {
						c.Wait()
					}
					c.L.Unlock()
					err = dockerClient.SwarmJoin(fmt.Sprintf("%s:2377", firstSwarmManager.IP), tokens.Worker)
					if err != nil {
						log.Printf("Cannot join worker %s to swarm. Got: %v\n", i.Name, err)
						return err
					}
				}
			}

			for _, cmd := range conf.Run {
				errch := make(chan error)
				go func() {
					exitCode, err := p.InstanceExec(i, cmd)
					fmt.Printf("Finished execuing command [%s] on instance %s with code [%d] and err [%v]\n", cmd, i.Name, exitCode, err)

					if err != nil {
						errch <- err
					}
					if exitCode != 0 {
						errch <- fmt.Errorf("Command returned %d on instance %s", exitCode, i.IP)
					}
					errch <- nil
				}()

				// ctx.Done() could be called if the errgroup is cancelled due to a previous error. In that case, return immediately
				select {
				case err = <-errch:
					return err
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
