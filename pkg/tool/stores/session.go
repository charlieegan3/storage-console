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
func (sdb *SessionDB) GetSession(sessionID string) (*webauthn.SessionData, bool, error) {

	var session struct {
		ID            string `db:"id"`
		SessionData   []byte `db:"session_data"`
		Authenticated bool   `db:"authenticated"`
	}

	_, err := sdb.db.From("curry_club.sessions").Select("id", "session_data", "authenticated").
		Where(goqu.Ex{"id": sessionID}).
		ScanStruct(&session)
	if err != nil {
		return nil, false, fmt.Errorf("error getting session '%s': %s", sessionID, err.Error())
	}

	var sessionData webauthn.SessionData
	err = json.Unmarshal(session.SessionData, &sessionData)
	if err != nil {
		return nil, false, fmt.Errorf("error unmarshalling session data: %s", err.Error())
	}

	return &sessionData, session.Authenticated, nil
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

func (sdb *SessionDB) StartSession(data *webauthn.SessionData) (string, error) {

	sessionData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("error marshalling session data: %s", err.Error())
	}

	var sessionId string
	_, err = sdb.db.Insert("curry_club.sessions").
		Rows(goqu.Record{
			"session_data": sessionData,
		}).Returning("id").Executor().ScanVal(&sessionId)
	if err != nil {
		return "", fmt.Errorf("error inserting session: %s", err.Error())
	}

	return sessionId, nil
}

func (sdb *SessionDB) AuthenticateSession(sessionID string) error {

	res, err := sdb.db.Update("curry_club.sessions").
		Where(goqu.Ex{"id": sessionID}).
		Set(
			goqu.Record{
				"authenticated": true,
			},
		).Executor().Exec()
	if err != nil {
		return fmt.Errorf("error authenticating session: %s", err.Error())
	}
	rowsAf, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting session affected: %s", err.Error())
	}
	if rowsAf == 0 {
		return fmt.Errorf("no session updated")
	}

	return nil
}
