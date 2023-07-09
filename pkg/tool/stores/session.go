package stores

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/go-webauthn/webauthn/webauthn"
)

type SessionDB struct {
	db *goqu.Database
}

func NewSessionDB(db *sql.DB) *SessionDB {

	goquDB := goqu.New("postgres", db)
	return &SessionDB{
		db: goquDB,
	}
}

// GetSession returns a *SessionData by the session's ID
func (sdb *SessionDB) GetSession(sessionID uint64) (*webauthn.SessionData, error) {

	var session struct {
		ID          uint64 `db:"id"`
		SessionData []byte `db:"session_data"`
	}

	_, err := sdb.db.From("curry_club.sessions").Select("id", "session_data").
		Where(goqu.Ex{"id": sessionID}).
		ScanStruct(&session)
	if err != nil {
		return nil, fmt.Errorf("error getting session '%s': %s", sessionID, err.Error())
	}

	var sessionData webauthn.SessionData
	err = json.Unmarshal(session.SessionData, &sessionData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling session data: %s", err.Error())
	}

	return &sessionData, nil
}

func (sdb *SessionDB) DeleteSession(sessionID string) error {

	_, err := sdb.db.Delete("curry_club.sessions").
		Where(goqu.Ex{"id": sessionID}).
		Executor().Exec()
	if err != nil {
		return fmt.Errorf("error deleting session '%s': %s", sessionID, err.Error())
	}

	return nil
}

func (sdb *SessionDB) StartSession(data *webauthn.SessionData) (uint64, error) {

	sessionData, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("error marshalling session data: %s", err.Error())
	}

	var sessionId uint64
	_, err = sdb.db.Insert("curry_club.sessions").
		Rows(goqu.Record{
			"session_data": sessionData,
		}).Returning("id").Executor().ScanVal(&sessionId)
	if err != nil {
		return 0, fmt.Errorf("error inserting session: %s", err.Error())
	}

	return sessionId, nil
}
