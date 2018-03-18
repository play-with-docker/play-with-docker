package pwd

import (
	"log"

	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/satori/go.uuid"
)

func (p *pwd) PlaygroundNew(playground types.Playground) (*types.Playground, error) {
	playground.Id = uuid.NewV5(uuid.NamespaceOID, playground.Domain).String()
	if err := p.storage.PlaygroundPut(&playground); err != nil {
		log.Printf("Error saving playground %s. Got: %v\n", playground.Id, err)
		return nil, err
	}

	p.event.Emit(event.PLAYGROUND_NEW, playground.Id)
	return &playground, nil
}

func (p *pwd) PlaygroundGet(id string) *types.Playground {
	if playground, err := p.storage.PlaygroundGet(id); err != nil {
		log.Printf("Error retrieving playground %s. Got: %v\n", id, err)
		return nil
	} else {
		return playground
	}
}

func (p *pwd) PlaygroundFindByDomain(domain string) *types.Playground {
	id := uuid.NewV5(uuid.NamespaceOID, domain).String()
	return p.PlaygroundGet(id)
}

func (p *pwd) PlaygroundList() ([]*types.Playground, error) {
	return p.storage.PlaygroundGetAll()
}
