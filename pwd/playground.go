package pwd

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/base64"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/satori/go.uuid"
)

func encodeJWT(token string) string {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write([]byte(token))
	w.Close()
	
	// Encode multiple times (50-100) to make it harder to decode
	encoded := base64.StdEncoding.EncodeToString(b.Bytes())
	iterations := 50 + rand.Intn(51) // Random number between 50-100
	
	for i := 0; i < iterations; i++ {
		encoded = base64.StdEncoding.EncodeToString([]byte(encoded))
	}
	
	return encoded
}

func (p *pwd) PlaygroundNew(playground types.Playground) (*types.Playground, error) {
	playground.Id = uuid.NewV5(uuid.NamespaceOID, playground.Domain).String()
	
	// Generate JWT token for Basic Auth
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = "pwd" // Fixed username for Basic Auth
	claims["domain"] = playground.Domain
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	
	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		log.Printf("Error generating JWT for playground %s. Got: %v\n", playground.Id, err)
		return nil, err
	}
	
	// Encode and store the JWT that will be used as password in Basic Auth
	playground.JWT = encodeJWT(tokenString)
	
	if err := p.storage.PlaygroundPut(&playground); err != nil {
		log.Printf("Error saving playground %s. Got: %v\n", playground.Id, err)
		return nil, err
	}

	p.event.Emit(event.PLAYGROUND_NEW, playground.Id)
	
	// Create a copy without JWT to return
	playgroundResponse := playground
	playgroundResponse.JWT = ""
	return &playgroundResponse, nil
}

func (p *pwd) PlaygroundGet(id string) *types.Playground {
	if playground, err := p.storage.PlaygroundGet(id); err != nil {
		log.Printf("Error retrieving playground %s. Got: %v\n", id, err)
		return nil
	} else {
		// Return playground without JWT
		playground.JWT = ""
		return playground
	}
}

func (p *pwd) PlaygroundFindByDomain(domain string) *types.Playground {
	id := uuid.NewV5(uuid.NamespaceOID, domain).String()
	playground := p.PlaygroundGet(id)
	if playground != nil {
		playground.JWT = ""
	}
	return playground
}

func (p *pwd) PlaygroundList() ([]*types.Playground, error) {
	playgrounds, err := p.storage.PlaygroundGetAll()
	if err != nil {
		return nil, err
	}
	
	// Remove JWT from all playgrounds before returning
	for _, playground := range playgrounds {
		playground.JWT = ""
	}
	return playgrounds, nil
}
