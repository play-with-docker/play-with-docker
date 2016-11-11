package services

import (
	"log"
	"math"
	"sync"
	"time"

	"github.com/googollee/go-socket.io"
	"github.com/twinj/uuid"
)

var wsServer *socketio.Server

type Session struct {
	sync.Mutex
	Id        string               `json:"id"`
	Instances map[string]*Instance `json:"instances"`
	Clients   []*Client            `json:"-"`
}

func (s *Session) GetSmallestViewPort() ViewPort {
	minRows := s.Clients[0].ViewPort.Rows
	minCols := s.Clients[0].ViewPort.Cols

	for _, c := range s.Clients {
		minRows = uint(math.Min(float64(minRows), float64(c.ViewPort.Rows)))
		minCols = uint(math.Min(float64(minCols), float64(c.ViewPort.Cols)))
	}

	return ViewPort{Rows: minRows, Cols: minCols}
}

func (s *Session) AddNewClient(c *Client) {
	s.Clients = append(s.Clients, c)
}

var sessions map[string]*Session

func init() {
	sessions = make(map[string]*Session)
}

func CreateWSServer() *socketio.Server {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	wsServer = server
	return server
}

func NewSession() (*Session, error) {
	s := &Session{}
	s.Id = uuid.NewV4().String()
	s.Instances = map[string]*Instance{}
	log.Printf("NewSession id=[%s]\n", s.Id)

	sessions[s.Id] = s

	// Schedule cleanup of the session
	time.AfterFunc(4*time.Hour, func() {
		s = GetSession(s.Id)
		s.Lock()
		defer s.Unlock()
		wsServer.BroadcastTo(s.Id, "session end")
		log.Printf("Starting clean up of session [%s]\n", s.Id)
		for _, i := range s.Instances {
			i.Conn.Close()
			if err := DeleteContainer(i.Name); err != nil {
				log.Println(err)
			}
		}
		if err := DeleteNetwork(s.Id); err != nil {
			log.Println(err)
		}
		delete(sessions, s.Id)
		log.Printf("Cleaned up session [%s]\n", s.Id)
	})

	if err := CreateNetwork(s.Id); err != nil {
		log.Println("ERROR NETWORKING")
		return nil, err
	}

	return s, nil
}

func GetSession(sessionId string) *Session {
	return sessions[sessionId]
}
