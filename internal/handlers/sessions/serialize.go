package sessions

import (
	"bytes"
	"encoding/json"
	"log"
)

type serializedSession struct {
	Id        string `json:"id"`
	CsrfToken string `json:"csrfToken"`
}

func serializeSession(s Session) ([]byte, error) {
	ss := serializedSession{
		Id:        s.ID,
		CsrfToken: s.CsrfToken,
	}

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(ss); err != nil {
		log.Printf("Could not serialize session %v: %v", s, err)
		return nil, err
	}

	return b.Bytes(), nil
}

func deserializeSession(b []byte) (Session, error) {
	var ss serializedSession
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&ss); err != nil {
		log.Printf("Failed to deserialize session from json: %v", err)
		return Session{}, err
	}

	return Session{
		ID:        ss.Id,
		CsrfToken: ss.CsrfToken,
	}, nil
}
