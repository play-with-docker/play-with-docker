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

	var loadedSessions map[string]*types.Session
	expectedSessions := map[string]*types.Session{}
	expectedSessions[s.Id] = s

	file, err := os.Open(tmpfile.Name())

	assert.Nil(t, err)
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&loadedSessions)

	assert.Nil(t, err)

	assert.EqualValues(t, expectedSessions, loadedSessions)
}

func TestSessionGet(t *testing.T) {
	expectedSession := &types.Session{Id: "session1"}
	sessions := map[string]*types.Session{}
	sessions[expectedSession.Id] = expectedSession

	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(&sessions)
	assert.Nil(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	_, err = storage.SessionGet("bad id")
	assert.True(t, NotFound(err))

	loadedSession, err := storage.SessionGet("session1")
	assert.Nil(t, err)

	assert.Equal(t, expectedSession, loadedSession)
}

func TestInstanceFindByIP(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	i1 := &types.Instance{Name: "i1", IP: "10.0.0.1"}
	i2 := &types.Instance{Name: "i2", IP: "10.1.0.1"}
	s1 := &types.Session{Id: "session1", Instances: map[string]*types.Instance{"i1": i1}}
	s2 := &types.Session{Id: "session2", Instances: map[string]*types.Instance{"i2": i2}}
	err = storage.SessionPut(s1)
	assert.Nil(t, err)
	err = storage.SessionPut(s2)
	assert.Nil(t, err)

	foundInstance, err := storage.InstanceFindByIP("10.0.0.1")
	assert.Nil(t, err)
	assert.Equal(t, i1, foundInstance)

	foundInstance, err = storage.InstanceFindByIP("10.1.0.1")
	assert.Nil(t, err)
	assert.Equal(t, i2, foundInstance)

	foundInstance, err = storage.InstanceFindByIP("192.168.0.1")
	assert.True(t, NotFound(err))
	assert.Nil(t, foundInstance)
}

func TestInstanceFindByIPAndSession(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	i1 := &types.Instance{Name: "i1", IP: "10.0.0.1"}
	i2 := &types.Instance{Name: "i2", IP: "10.1.0.1"}
	s1 := &types.Session{Id: "session1", Instances: map[string]*types.Instance{"i1": i1}}
	s2 := &types.Session{Id: "session2", Instances: map[string]*types.Instance{"i2": i2}}
	err = storage.SessionPut(s1)
	assert.Nil(t, err)
	err = storage.SessionPut(s2)
	assert.Nil(t, err)

	foundInstance, err := storage.InstanceFindByIPAndSession("session1", "10.0.0.1")
	assert.Nil(t, err)
	assert.Equal(t, i1, foundInstance)

	foundInstance, err = storage.InstanceFindByIPAndSession("session2", "10.1.0.1")
	assert.Nil(t, err)
	assert.Equal(t, i2, foundInstance)

	foundInstance, err = storage.InstanceFindByIPAndSession("session3", "10.1.0.1")
	assert.True(t, NotFound(err))
	assert.Nil(t, foundInstance)

	foundInstance, err = storage.InstanceFindByIPAndSession("session1", "10.1.0.1")
	assert.True(t, NotFound(err))
	assert.Nil(t, foundInstance)

	foundInstance, err = storage.InstanceFindByIPAndSession("session1", "192.168.0.1")
	assert.True(t, NotFound(err))
	assert.Nil(t, foundInstance)
}

func TestInstanceFindByAlias(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	i1 := &types.Instance{Name: "i1", Alias: "foo", IP: "10.0.0.1"}
	i2 := &types.Instance{Name: "i2", Alias: "foo", IP: "10.1.0.1"}
	s1 := &types.Session{Id: "session1", Instances: map[string]*types.Instance{"i1": i1}}
	s2 := &types.Session{Id: "session2", Instances: map[string]*types.Instance{"i2": i2}}
	err = storage.SessionPut(s1)
	assert.Nil(t, err)
	err = storage.SessionPut(s2)
	assert.Nil(t, err)

	foundInstance, err := storage.InstanceFindByAlias("session1", "foo")
	assert.Nil(t, err)
	assert.Equal(t, i1, foundInstance)

	foundInstance, err = storage.InstanceFindByAlias("session2", "foo")
	assert.Nil(t, err)
	assert.Equal(t, i2, foundInstance)

	foundInstance, err = storage.InstanceFindByAlias("session1", "bar")
	assert.True(t, NotFound(err))
	assert.Nil(t, foundInstance)

	foundInstance, err = storage.InstanceFindByAlias("session3", "foo")
	assert.True(t, NotFound(err))
	assert.Nil(t, foundInstance)
}

func TestCounts(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	defer os.Remove(tmpfile.Name())

	storage, err := NewFileStorage(tmpfile.Name())

	assert.Nil(t, err)

	c1 := &types.Client{}
	i1 := &types.Instance{Name: "i1", Alias: "foo", IP: "10.0.0.1"}
	i2 := &types.Instance{Name: "i2", Alias: "foo", IP: "10.1.0.1"}
	s1 := &types.Session{Id: "session1", Instances: map[string]*types.Instance{"i1": i1}}
	s2 := &types.Session{Id: "session2", Instances: map[string]*types.Instance{"i2": i2}}
	s3 := &types.Session{Id: "session3", Clients: []*types.Client{c1}}

	err = storage.SessionPut(s1)
	assert.Nil(t, err)
	err = storage.SessionPut(s2)
	assert.Nil(t, err)
	err = storage.SessionPut(s3)
	assert.Nil(t, err)

	num, err := storage.SessionCount()
	assert.Nil(t, err)
	assert.Equal(t, 3, num)

	num, err = storage.InstanceCount()
	assert.Nil(t, err)
	assert.Equal(t, 2, num)

	num, err = storage.ClientCount()
	assert.Nil(t, err)
	assert.Equal(t, 1, num)
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
