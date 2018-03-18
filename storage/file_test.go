package storage

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/assert"
)

func TestSessionPut(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	s := &types.Session{Id: "a session"}
	err = storage.SessionPut(s)

	assert.Nil(t, err)

	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{s.Id: s},
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
	var loadedDB *DB

	file, err := os.Open(tmpfile.Name())

	assert.Nil(t, err)
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&loadedDB)

	assert.Nil(t, err)

	assert.EqualValues(t, expectedDB, loadedDB)
}

func TestSessionGet(t *testing.T) {
	expectedSession := &types.Session{Id: "aaabbbccc"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{expectedSession.Id: expectedSession},
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

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	_, err = storage.SessionGet("foobar")
	assert.True(t, NotFound(err))

	loadedSession, err := storage.SessionGet("aaabbbccc")
	assert.Nil(t, err)

	assert.Equal(t, expectedSession, loadedSession)
}

func TestSessionGetAll(t *testing.T) {
	s1 := &types.Session{Id: "aaabbbccc"}
	s2 := &types.Session{Id: "dddeeefff"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{s1.Id: s1, s2.Id: s2},
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

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	sessions, err := storage.SessionGetAll()
	assert.Nil(t, err)

	assert.Subset(t, sessions, []*types.Session{s1, s2})
	assert.Len(t, sessions, 2)
}

func TestSessionDelete(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	s1 := &types.Session{Id: "session1"}
	err = storage.SessionPut(s1)
	assert.Nil(t, err)

	found, err := storage.SessionGet(s1.Id)
	assert.Nil(t, err)
	assert.Equal(t, s1, found)

	err = storage.SessionDelete(s1.Id)
	assert.Nil(t, err)

	found, err = storage.SessionGet(s1.Id)
	assert.True(t, NotFound(err))
	assert.Nil(t, found)
}

func TestInstanceGet(t *testing.T) {
	expectedInstance := &types.Instance{SessionId: "aaabbbccc", Name: "i1", IP: "10.0.0.1"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{},
		Instances:                   map[string]*types.Instance{expectedInstance.Name: expectedInstance},
		Clients:                     map[string]*types.Client{},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{expectedInstance.SessionId: []string{expectedInstance.Name}},
		ClientsBySessionId:          map[string][]string{},
		UsersByProvider:             map[string]string{},
	}

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	foundInstance, err := storage.InstanceGet("i1")
	assert.Nil(t, err)
	assert.Equal(t, expectedInstance, foundInstance)
}

func TestInstancePut(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	s := &types.Session{Id: "aaabbbccc"}
	i := &types.Instance{Name: "i1", IP: "10.0.0.1", SessionId: s.Id}

	err = storage.SessionPut(s)
	assert.Nil(t, err)

	err = storage.InstancePut(i)
	assert.Nil(t, err)

	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{s.Id: s},
		Instances:                   map[string]*types.Instance{i.Name: i},
		Clients:                     map[string]*types.Client{},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{i.SessionId: []string{i.Name}},
		ClientsBySessionId:          map[string][]string{},
		UsersByProvider:             map[string]string{},
	}
	var loadedDB *DB

	file, err := os.Open(tmpfile.Name())

	assert.Nil(t, err)
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&loadedDB)

	assert.Nil(t, err)

	assert.EqualValues(t, expectedDB, loadedDB)
}

func TestInstanceDelete(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	s := &types.Session{Id: "session1"}
	err = storage.SessionPut(s)
	assert.Nil(t, err)

	i := &types.Instance{Name: "i1", IP: "10.0.0.1", SessionId: s.Id}
	err = storage.InstancePut(i)
	assert.Nil(t, err)

	found, err := storage.InstanceGet(i.Name)
	assert.Nil(t, err)
	assert.Equal(t, i, found)

	err = storage.InstanceDelete(i.Name)
	assert.Nil(t, err)

	found, err = storage.InstanceGet(i.Name)
	assert.True(t, NotFound(err))
	assert.Nil(t, found)
}

func TestInstanceFindBySessionId(t *testing.T) {
	i1 := &types.Instance{SessionId: "aaabbbccc", Name: "c1"}
	i2 := &types.Instance{SessionId: "aaabbbccc", Name: "c2"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{},
		Instances:                   map[string]*types.Instance{i1.Name: i1, i2.Name: i2},
		Clients:                     map[string]*types.Client{},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{i1.SessionId: []string{i1.Name, i2.Name}},
		ClientsBySessionId:          map[string][]string{},
		UsersByProvider:             map[string]string{},
	}

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	instances, err := storage.InstanceFindBySessionId("aaabbbccc")
	assert.Nil(t, err)
	assert.Subset(t, instances, []*types.Instance{i1, i2})
	assert.Len(t, instances, 2)
}

func TestWindowsInstanceGetAll(t *testing.T) {
	i1 := &types.WindowsInstance{SessionId: "aaabbbccc", Id: "i1"}
	i2 := &types.WindowsInstance{SessionId: "aaabbbccc", Id: "i2"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{},
		Instances:                   map[string]*types.Instance{},
		Clients:                     map[string]*types.Client{},
		WindowsInstances:            map[string]*types.WindowsInstance{i1.Id: i1, i2.Id: i2},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{},
		WindowsInstancesBySessionId: map[string][]string{i1.SessionId: []string{i1.Id, i2.Id}},
		InstancesBySessionId:        map[string][]string{},
		ClientsBySessionId:          map[string][]string{},
		UsersByProvider:             map[string]string{},
	}

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	instances, err := storage.WindowsInstanceGetAll()
	assert.Nil(t, err)
	assert.Subset(t, instances, []*types.WindowsInstance{i1, i2})
	assert.Len(t, instances, 2)
}

func TestWindowsInstancePut(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	s := &types.Session{Id: "aaabbbccc"}
	i := &types.WindowsInstance{Id: "i1", SessionId: s.Id}

	err = storage.SessionPut(s)
	assert.Nil(t, err)

	err = storage.WindowsInstancePut(i)
	assert.Nil(t, err)

	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{s.Id: s},
		Instances:                   map[string]*types.Instance{},
		Clients:                     map[string]*types.Client{},
		WindowsInstances:            map[string]*types.WindowsInstance{i.Id: i},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{},
		WindowsInstancesBySessionId: map[string][]string{i.SessionId: []string{i.Id}},
		InstancesBySessionId:        map[string][]string{},
		ClientsBySessionId:          map[string][]string{},
		UsersByProvider:             map[string]string{},
	}
	var loadedDB *DB

	file, err := os.Open(tmpfile.Name())

	assert.Nil(t, err)
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&loadedDB)

	assert.Nil(t, err)

	assert.EqualValues(t, expectedDB, loadedDB)
}

func TestWindowsInstanceDelete(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	s := &types.Session{Id: "session1"}
	err = storage.SessionPut(s)
	assert.Nil(t, err)

	i := &types.WindowsInstance{Id: "i1", SessionId: s.Id}
	err = storage.WindowsInstancePut(i)
	assert.Nil(t, err)

	found, err := storage.WindowsInstanceGetAll()
	assert.Nil(t, err)
	assert.Equal(t, []*types.WindowsInstance{i}, found)

	err = storage.WindowsInstanceDelete(i.Id)
	assert.Nil(t, err)

	found, err = storage.WindowsInstanceGetAll()
	assert.Nil(t, err)
	assert.Empty(t, found)
}

func TestClientGet(t *testing.T) {
	c := &types.Client{SessionId: "aaabbbccc", Id: "c1"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{},
		Instances:                   map[string]*types.Instance{},
		Clients:                     map[string]*types.Client{c.Id: c},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{},
		ClientsBySessionId:          map[string][]string{c.SessionId: []string{c.Id}},
		UsersByProvider:             map[string]string{},
	}

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	found, err := storage.ClientGet("c1")
	assert.Nil(t, err)
	assert.Equal(t, c, found)
}

func TestClientPut(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	s := &types.Session{Id: "aaabbbccc"}
	c := &types.Client{Id: "c1", SessionId: s.Id}

	err = storage.SessionPut(s)
	assert.Nil(t, err)

	err = storage.ClientPut(c)
	assert.Nil(t, err)

	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{s.Id: s},
		Instances:                   map[string]*types.Instance{},
		Clients:                     map[string]*types.Client{c.Id: c},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{},
		ClientsBySessionId:          map[string][]string{c.SessionId: []string{c.Id}},
		UsersByProvider:             map[string]string{},
	}
	var loadedDB *DB

	file, err := os.Open(tmpfile.Name())

	assert.Nil(t, err)
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&loadedDB)

	assert.Nil(t, err)

	assert.EqualValues(t, expectedDB, loadedDB)
}

func TestClientDelete(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	s := &types.Session{Id: "session1"}
	err = storage.SessionPut(s)
	assert.Nil(t, err)

	c := &types.Client{Id: "c1", SessionId: s.Id}
	err = storage.ClientPut(c)
	assert.Nil(t, err)

	found, err := storage.ClientGet(c.Id)
	assert.Nil(t, err)
	assert.Equal(t, c, found)

	err = storage.ClientDelete(c.Id)
	assert.Nil(t, err)

	found, err = storage.ClientGet(c.Id)
	assert.True(t, NotFound(err))
	assert.Nil(t, found)
}

func TestClientFindBySessionId(t *testing.T) {
	c1 := &types.Client{SessionId: "aaabbbccc", Id: "c1"}
	c2 := &types.Client{SessionId: "aaabbbccc", Id: "c2"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{},
		Instances:                   map[string]*types.Instance{},
		Clients:                     map[string]*types.Client{c1.Id: c1, c2.Id: c2},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{},
		ClientsBySessionId:          map[string][]string{c1.SessionId: []string{c1.Id, c2.Id}},
		UsersByProvider:             map[string]string{},
	}

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	clients, err := storage.ClientFindBySessionId("aaabbbccc")
	assert.Nil(t, err)
	assert.Subset(t, clients, []*types.Client{c1, c2})
	assert.Len(t, clients, 2)
}

func TestPlaygroundGet(t *testing.T) {
	p := &types.Playground{Id: "aaabbbccc"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{},
		Instances:                   map[string]*types.Instance{},
		Clients:                     map[string]*types.Client{},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{p.Id: p},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{},
		ClientsBySessionId:          map[string][]string{},
		UsersByProvider:             map[string]string{},
	}

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	found, err := storage.PlaygroundGet("aaabbbccc")
	assert.Nil(t, err)
	assert.Equal(t, p, found)
}

func TestPlaygroundPut(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	p := &types.Playground{Id: "aaabbbccc"}

	err = storage.PlaygroundPut(p)
	assert.Nil(t, err)

	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{},
		Instances:                   map[string]*types.Instance{},
		Clients:                     map[string]*types.Client{},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{p.Id: p},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{},
		ClientsBySessionId:          map[string][]string{},
		UsersByProvider:             map[string]string{},
	}
	var loadedDB *DB

	file, err := os.Open(tmpfile.Name())

	assert.Nil(t, err)
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&loadedDB)

	assert.Nil(t, err)

	assert.EqualValues(t, expectedDB, loadedDB)
}

func TestPlaygroundGetAll(t *testing.T) {
	p1 := &types.Playground{Id: "aaabbbccc"}
	p2 := &types.Playground{Id: "dddeeefff"}
	expectedDB := &DB{
		Sessions:                    map[string]*types.Session{},
		Instances:                   map[string]*types.Instance{},
		Clients:                     map[string]*types.Client{},
		WindowsInstances:            map[string]*types.WindowsInstance{},
		LoginRequests:               map[string]*types.LoginRequest{},
		Users:                       map[string]*types.User{},
		Playgrounds:                 map[string]*types.Playground{p1.Id: p1, p2.Id: p2},
		WindowsInstancesBySessionId: map[string][]string{},
		InstancesBySessionId:        map[string][]string{},
		ClientsBySessionId:          map[string][]string{},
		UsersByProvider:             map[string]string{},
	}

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&expectedDB)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	found, err := storage.PlaygroundGetAll()
	assert.Nil(t, err)
	assert.Subset(t, []*types.Playground{p1, p2}, found)
	assert.Len(t, found, 2)
}
