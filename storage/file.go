package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type storage struct {
	rw   sync.Mutex
	path string
	db   map[string]*types.Session
}

func (store *storage) SessionGet(sessionId string) (*types.Session, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	s, found := store.db[sessionId]
	if !found {
		return nil, fmt.Errorf("%s", notFound)
	}

	return s, nil
}
func (store *storage) SessionPut(s *types.Session) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	store.db[s.Id] = s

	return store.save()
}

func (store *storage) InstanceFindByIP(ip string) (*types.Instance, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	for _, s := range store.db {
		for _, i := range s.Instances {
			if i.IP == ip {
				return i, nil
			}
		}
	}

	return nil, fmt.Errorf("%s", notFound)
}

func (store *storage) InstanceFindByIPAndSession(sessionPrefix, ip string) (*types.Instance, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	for id, s := range store.db {
		if strings.HasPrefix(id, sessionPrefix) {
			for _, i := range s.Instances {
				if i.IP == ip {
					return i, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("%s", notFound)
}

func (store *storage) InstanceFindByAlias(sessionPrefix, alias string) (*types.Instance, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	for id, s := range store.db {
		if strings.HasPrefix(id, sessionPrefix) {
			for _, i := range s.Instances {
				if i.Alias == alias {
					return i, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("%s", notFound)
}

func (store *storage) SessionCount() (int, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	return len(store.db), nil
}

func (store *storage) InstanceCount() (int, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	var ins int

	for _, s := range store.db {
		ins += len(s.Instances)
	}

	return ins, nil
}

func (store *storage) ClientCount() (int, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	var cli int

	for _, s := range store.db {
		cli += len(s.Clients)
	}

	return cli, nil
}

func (store *storage) SessionDelete(sessionId string) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	delete(store.db, sessionId)
	return store.save()
}

func (store *storage) load() error {
	file, err := os.Open(store.path)

	if err == nil {
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&store.db)

		if err != nil {
			return err
		}
	} else {
		store.db = map[string]*types.Session{}
	}

	file.Close()
	return nil
}

func (store *storage) save() error {
	file, err := os.Create(store.path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(&store.db)
	return err
}

func NewFileStorage(path string) (StorageApi, error) {
	s := &storage{path: path}

	err := s.load()
	if err != nil {
		return nil, err
	}

	return s, nil
}
