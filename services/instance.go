package services

import (
	"context"
	"io"
	"log"
	"os"

	"golang.org/x/text/encoding"

	"github.com/docker/docker/api/types"
)

type Instance struct {
	Session *Session                `json:"-"`
	Name    string                  `json:"name"`
	IP      string                  `json:"ip"`
	Conn    *types.HijackedResponse `json:"-"`
	ExecId  string                  `json:"-"`
	Ctx     context.Context         `json:"-"`
}

var dindImage string
var defaultDindImageName string

func init() {
	dindImage = getDindImageName()
}

func getDindImageName() string {
	dindImage := os.Getenv("DIND_IMAGE")
	defaultDindImageName = "docker:1.12.2-rc2-dind"
	if len(dindImage) == 0 {
		dindImage = defaultDindImageName
	}
	return dindImage
}

func NewInstance(session *Session) (*Instance, error) {
	log.Printf("NewInstance - using image: [%s]\n", dindImage)
	instance, err := CreateInstance(session.Id, dindImage)
	if err != nil {
		return nil, err
	}
	instance.Session = session

	if session.Instances == nil {
		session.Instances = make(map[string]*Instance)
	}
	session.Instances[instance.Name] = instance

	go instance.Exec()

	wsServer.BroadcastTo(session.Id, "new instance", instance.Name, instance.IP)

	return instance, nil
}

type sessionWriter struct {
	instance *Instance
}

func (s *sessionWriter) Write(p []byte) (n int, err error) {
	wsServer.BroadcastTo(s.instance.Session.Id, "terminal out", s.instance.Name, string(p))
	return len(p), nil
}

func (i *Instance) ResizeTerminal(cols, rows uint) error {
	return ResizeExecConnection(i.ExecId, i.Ctx, cols, rows)
}

func (i *Instance) Exec() {
	i.Ctx = context.Background()

	id, err := CreateExecConnection(i.Name, i.Ctx)
	if err != nil {
		return
	}
	i.ExecId = id
	conn, err := AttachExecConnection(id, i.Ctx)
	if err != nil {
		return
	}

	i.Conn = conn

	go func() {
		encoder := encoding.Replacement.NewEncoder()
		sw := &sessionWriter{instance: i}
		io.Copy(encoder.Writer(sw), conn.Reader)
	}()

	select {
	case <-i.Ctx.Done():
	}
}

func GetInstance(session *Session, name string) *Instance {
	//TODO: Use redis
	return session.Instances[name]
}
func DeleteInstance(session *Session, instance *Instance) error {
	//TODO: Use redis
	delete(session.Instances, instance.Name)
	err := DeleteContainer(instance.Name)

	wsServer.BroadcastTo(session.Id, "delete instance", instance.Name)

	return err
}
