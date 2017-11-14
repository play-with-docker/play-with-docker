package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type storage struct {
	rw   sync.Mutex
	path string
	db   *DB
}

type DB struct {
	Sessions         map[string]*types.Session         `json:"sessions"`
	Instances        map[string]*types.Instance        `json:"instances"`
	Clients          map[string]*types.Client          `json:"clients"`
	WindowsInstances map[string]*types.WindowsInstance `json:"windows_instances"`
	LoginRequests    map[string]*types.LoginRequest    `json:"login_requests"`
	Users            map[string]*types.User            `json:"user"`
	Playgrounds      map[string]*types.Playground      `json:"playgrounds"`

	WindowsInstancesBySessionId map[string][]string `json:"windows_instances_by_session_id"`
	InstancesBySessionId        map[string][]string `json:"instances_by_session_id"`
	ClientsBySessionId          map[string][]string `json:"clients_by_session_id"`
	UsersByProvider             map[string]string   `json:"users_by_providers"`
}

func (store *storage) SessionGet(id string) (*types.Session, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	s, found := store.db.Sessions[id]
	if !found {
		return nil, NotFoundError
	}

	return s, nil
}

func (store *storage) SessionGetAll() ([]*types.Session, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	sessions := make([]*types.Session, len(store.db.Sessions))
	i := 0
	for _, s := range store.db.Sessions {
		sessions[i] = s
		i++
	}

	return sessions, nil
}

func (store *storage) SessionPut(session *types.Session) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	store.db.Sessions[session.Id] = session

	return store.save()
}

func (store *storage) SessionDelete(id string) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	_, found := store.db.Sessions[id]
	if !found {
		return nil
	}
	for _, i := range store.db.WindowsInstancesBySessionId[id] {
		delete(store.db.WindowsInstances, i)
	}
	store.db.WindowsInstancesBySessionId[id] = []string{}
	for _, i := range store.db.InstancesBySessionId[id] {
		delete(store.db.Instances, i)
	}
	store.db.InstancesBySessionId[id] = []string{}
	for _, i := range store.db.ClientsBySessionId[id] {
		delete(store.db.Clients, i)
	}
	store.db.ClientsBySessionId[id] = []string{}
	delete(store.db.Sessions, id)

	return store.save()
}

func (store *storage) SessionCount() (int, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	return len(store.db.Sessions), nil
}

func (store *storage) InstanceGet(name string) (*types.Instance, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	i := store.db.Instances[name]
	if i == nil {
		return nil, NotFoundError
	}
	return i, nil
}

func (store *storage) InstancePut(instance *types.Instance) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	_, found := store.db.Sessions[string(instance.SessionId)]
	if !found {
		return NotFoundError
	}

	store.db.Instances[instance.Name] = instance
	found = false
	for _, i := range store.db.InstancesBySessionId[string(instance.SessionId)] {
		if i == instance.Name {
			found = true
			break
		}
	}
	if !found {
		store.db.InstancesBySessionId[string(instance.SessionId)] = append(store.db.InstancesBySessionId[string(instance.SessionId)], instance.Name)
	}

	return store.save()
}

func (store *storage) InstanceDelete(name string) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	instance, found := store.db.Instances[name]
	if !found {
		return nil
	}

	instances := store.db.InstancesBySessionId[string(instance.SessionId)]
	for n, i := range instances {
		if i == name {
			instances = append(instances[:n], instances[n+1:]...)
			break
		}
	}
	store.db.InstancesBySessionId[string(instance.SessionId)] = instances
	delete(store.db.Instances, name)

	return store.save()
}

func (store *storage) InstanceCount() (int, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	return len(store.db.Instances), nil
}

func (store *storage) InstanceFindBySessionId(sessionId string) ([]*types.Instance, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	instanceIds := store.db.InstancesBySessionId[sessionId]
	instances := make([]*types.Instance, len(instanceIds))
	for i, id := range instanceIds {
		instances[i] = store.db.Instances[id]
	}

	return instances, nil
}

func (store *storage) WindowsInstanceGetAll() ([]*types.WindowsInstance, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	instances := []*types.WindowsInstance{}

	for _, s := range store.db.WindowsInstances {
		instances = append(instances, s)
	}

	return instances, nil
}

func (store *storage) WindowsInstancePut(instance *types.WindowsInstance) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	_, found := store.db.Sessions[string(instance.SessionId)]
	if !found {
		return NotFoundError
	}
	store.db.WindowsInstances[instance.Id] = instance
	found = false
	for _, i := range store.db.WindowsInstancesBySessionId[string(instance.SessionId)] {
		if i == instance.Id {
			found = true
			break
		}
	}
	if !found {
		store.db.WindowsInstancesBySessionId[string(instance.SessionId)] = append(store.db.WindowsInstancesBySessionId[string(instance.SessionId)], instance.Id)
	}

	return store.save()
}

func (store *storage) WindowsInstanceDelete(id string) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	instance, found := store.db.WindowsInstances[id]
	if !found {
		return nil
	}

	instances := store.db.WindowsInstancesBySessionId[string(instance.SessionId)]
	for n, i := range instances {
		if i == id {
			instances = append(instances[:n], instances[n+1:]...)
			break
		}
	}
	store.db.WindowsInstancesBySessionId[string(instance.SessionId)] = instances
	delete(store.db.WindowsInstances, id)

	return store.save()
}

func (store *storage) ClientGet(id string) (*types.Client, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	i := store.db.Clients[id]
	if i == nil {
		return nil, NotFoundError
	}
	return i, nil
}
func (store *storage) ClientPut(client *types.Client) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	_, found := store.db.Sessions[string(client.SessionId)]
	if !found {
		return NotFoundError
	}

	store.db.Clients[client.Id] = client
	found = false
	for _, i := range store.db.ClientsBySessionId[string(client.SessionId)] {
		if i == client.Id {
			found = true
			break
		}
	}
	if !found {
		store.db.ClientsBySessionId[string(client.SessionId)] = append(store.db.ClientsBySessionId[string(client.SessionId)], client.Id)
	}

	return store.save()
}
func (store *storage) ClientDelete(id string) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	client, found := store.db.Clients[id]
	if !found {
		return nil
	}

	clients := store.db.ClientsBySessionId[string(client.SessionId)]
	for n, i := range clients {
		if i == client.Id {
			clients = append(clients[:n], clients[n+1:]...)
			break
		}
	}
	store.db.ClientsBySessionId[string(client.SessionId)] = clients
	delete(store.db.Clients, id)

	return store.save()
}
func (store *storage) ClientCount() (int, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	return len(store.db.Clients), nil
}
func (store *storage) ClientFindBySessionId(sessionId string) ([]*types.Client, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	clientIds := store.db.ClientsBySessionId[sessionId]
	clients := make([]*types.Client, len(clientIds))
	for i, id := range clientIds {
		clients[i] = store.db.Clients[id]
	}

	return clients, nil
}

func (store *storage) LoginRequestPut(loginRequest *types.LoginRequest) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	store.db.LoginRequests[loginRequest.Id] = loginRequest
	return nil
}
func (store *storage) LoginRequestGet(id string) (*types.LoginRequest, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	if lr, found := store.db.LoginRequests[id]; !found {
		return nil, NotFoundError
	} else {
		return lr, nil
	}
}
func (store *storage) LoginRequestDelete(id string) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	delete(store.db.LoginRequests, id)
	return nil
}

func (store *storage) UserFindByProvider(providerName, providerUserId string) (*types.User, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	if userId, found := store.db.UsersByProvider[fmt.Sprintf("%s_%s", providerName, providerUserId)]; !found {
		return nil, NotFoundError
	} else {
		if user, found := store.db.Users[userId]; !found {
			return nil, NotFoundError
		} else {
			return user, nil
		}
	}
}

func (store *storage) UserPut(user *types.User) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	store.db.UsersByProvider[fmt.Sprintf("%s_%s", user.Provider, user.ProviderUserId)] = user.Id
	store.db.Users[user.Id] = user

	return store.save()
}
func (store *storage) UserGet(id string) (*types.User, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	if user, found := store.db.Users[id]; !found {
		return nil, NotFoundError
	} else {
		return user, nil
	}
}

func (store *storage) PlaygroundPut(playground *types.Playground) error {
	store.rw.Lock()
	defer store.rw.Unlock()

	store.db.Playgrounds[playground.Id] = playground

	return store.save()
}
func (store *storage) PlaygroundGet(id string) (*types.Playground, error) {
	store.rw.Lock()
	defer store.rw.Unlock()
	if playground, found := store.db.Playgrounds[id]; !found {
		return nil, NotFoundError
	} else {
		return playground, nil
	}
	return nil, NotFoundError
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
		store.db = &DB{
			Sessions:                    map[string]*types.Session{},
			Instances:                   map[string]*types.Instance{},
			Clients:                     map[string]*types.Client{},
			WindowsInstances:            map[string]*types.WindowsInstance{},
			LoginRequests:               map[string]*types.LoginRequest{},
			Users:                       map[string]*types.User{},
			Playgrounds:                 map[string]*types.Playground{},
			WindowsInstancesBySessionId: map[string][]string{},
			InstancesBySessionId:        map[string][]string{},
			ClientsBySessionId:          map[string][]string{},
			UsersByProvider:             map[string]string{},
		}
	}

	file.Close()
	return nil
}
func (store *storage) PlaygroundGetAll() ([]*types.Playground, error) {
	store.rw.Lock()
	defer store.rw.Unlock()

	playgrounds := make([]*types.Playground, len(store.db.Playgrounds))
	i := 0
	for _, p := range store.db.Playgrounds {
		playgrounds[i] = p
		i++
	}

	return playgrounds, nil
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
