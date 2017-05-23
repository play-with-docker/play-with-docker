package pwd

import (
	"encoding/gob"
	"os"
	"sync"

	"github.com/play-with-docker/play-with-docker/config"
)

type StorageApi interface {
	Save() error
	Load() error
}

type storage struct {
	rw sync.Mutex
}

func (store *storage) Load() error {
	file, err := os.Open(config.SessionsFile)

	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&sessions)

		if err != nil {
			return err
		}
	}

	file.Close()
	return nil
}

func (store *storage) Save() error {
	store.rw.Lock()
	defer store.rw.Unlock()
	file, err := os.Create(config.SessionsFile)
	if err == nil {
		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&sessions)
	}
	file.Close()
	return nil
}

func NewStorage() *storage {
	return &storage{}
}
