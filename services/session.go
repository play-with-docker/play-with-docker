package services

import (
	"encoding/gob"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"github.com/googollee/go-socket.io"
	"github.com/twinj/uuid"
)

var wsServer *socketio.Server

type Session struct {
	rw        sync.Mutex
	Id        string               `json:"id"`
	Instances map[string]*Instance `json:"instances"`
	clients   []*Client            `json:"-"`
	CreatedAt time.Time            `json:"created_at"`
	ExpiresAt time.Time            `json:"expires_at"`
}

func (s *Session) Lock() {
	s.rw.Lock()
}

func (s *Session) Unlock() {
	s.rw.Unlock()
}

func (s *Session) GetSmallestViewPort() ViewPort {
	minRows := s.clients[0].ViewPort.Rows
	minCols := s.clients[0].ViewPort.Cols

	for _, c := range s.clients {
		minRows = uint(math.Min(float64(minRows), float64(c.ViewPort.Rows)))
		minCols = uint(math.Min(float64(minCols), float64(c.ViewPort.Cols)))
	}

	return ViewPort{Rows: minRows, Cols: minCols}
}

func (s *Session) AddNewClient(c *Client) {
	s.clients = append(s.clients, c)
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

func CloseSessionAfter(s *Session, d time.Duration) {
	time.AfterFunc(d, func() {
		CloseSession(s)
	})
}

func CloseSession(s *Session) error {
	s.rw.Lock()
	defer s.rw.Unlock()
	wsServer.BroadcastTo(s.Id, "session end")
	log.Printf("Starting clean up of session [%s]\n", s.Id)
	for _, i := range s.Instances {
		if i.conn != nil {
			i.conn.Close()
		}
		if err := DeleteContainer(i.Name); err != nil {
			log.Println(err)
			return err
		}
	}
	if err := DeleteNetwork(s.Id); err != nil {
		log.Println(err)
		return err
	}
	delete(sessions, s.Id)
	log.Printf("Cleaned up session [%s]\n", s.Id)
	return nil
}

func NewSession() (*Session, error) {
	s := &Session{}
	s.Id = uuid.NewV4().String()
	s.Instances = map[string]*Instance{}
	s.CreatedAt = time.Now()
	s.ExpiresAt = s.CreatedAt.Add(4 * time.Hour)
	log.Printf("NewSession id=[%s]\n", s.Id)

	sessions[s.Id] = s

	// Schedule cleanup of the session
	CloseSessionAfter(s, 4*time.Hour)

	if err := CreateNetwork(s.Id); err != nil {
		log.Println("ERROR NETWORKING")
		return nil, err
	}

	// We store sessions as soon as we create one so we don't delete new sessions on an api restart
	if err := saveSessionsToDisk(); err != nil {
		return nil, err
	}
	return s, nil
}

func GetSession(sessionId string) *Session {
	s := sessions[sessionId]
	if s != nil {
		for _, instance := range s.Instances {
			if !instance.IsConnected() {
				instance.SetSession(s)
				go instance.Attach()
			}
		}

	}
	return s
}

func LoadSessionsFromDisk() error {
	file, err := os.Open("./pwd/sessions.gob")
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&sessions)

		if err != nil {
			return err
		}

		// schedule session expiration
		for _, s := range sessions {
			timeLeft := s.ExpiresAt.Sub(time.Now())
			CloseSessionAfter(s, timeLeft)

			// start collecting stats for every instance
			for _, i := range s.Instances {
				// wire the session back to the instance
				i.session = s
				go i.CollectStats()
			}
		}
	}
	file.Close()
	return err
}

func saveSessionsToDisk() error {
	rw.Lock()
	defer rw.Unlock()
	file, err := os.Create("./pwd/sessions.gob")
	if err == nil {
		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&sessions)
	}
	file.Close()
	return err
}
