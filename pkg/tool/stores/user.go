package stores

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/charlieegan3/curry-club/pkg/tool/types"
)

type UserDB struct {
	db *goqu.Database
}

func NewUsersDB(db *sql.DB) *UserDB {
	goquDB := goqu.New("postgres", db)

	return &UserDB{
		db: goquDB,
	}
}

func (udb *UserDB) GetUser(username string) (*types.User, error) {

	var user struct {
		ID          string `db:"id"`
		Username    string `db:"username"`
		Credentials []byte `db:"credentials"`
	}
	_, err := udb.db.From("curry_club.users").
		Where(goqu.Ex{"username": username}).
		ScanStruct(&user)
	if err != nil {
		return nil, fmt.Errorf("error getting user '%s': %s", username, err.Error())
	}

	var credentials []webauthn.Credential
	err = json.Unmarshal(user.Credentials, &credentials)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling credentials: %s", err.Error())
	}

	return &types.User{
		ID:          user.ID,
		Username:    user.Username,
		Credentials: credentials,
	}, nil
}

func (udb *UserDB) GetUserByID(id string) (*types.User, error) {

	var user struct {
		ID          string `db:"id"`
		Username    string `db:"username"`
		Credentials []byte `db:"credentials"`
	}
	_, err := udb.db.From("curry_club.users").
		Where(goqu.Ex{"id": id}).
		ScanStruct(&user)
	if err != nil {
		return nil, fmt.Errorf("error getting user '%d': %s", id, err.Error())
	}

	var credentials []webauthn.Credential
	err = json.Unmarshal(user.Credentials, &credentials)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling credentials: %s", err.Error())
	}

	return &types.User{
		ID:          user.ID,
		Username:    user.Username,
		Credentials: credentials,
	}, nil
}

func (udb *UserDB) AddCredentialsForUser(user *types.User, credentials []webauthn.Credential) error {

	if user.ID == "" {
		return fmt.Errorf("user has no ID")
	}

	user.Credentials = append(user.Credentials, credentials...)

	return udb.PutUser(user)
}

func (udb *UserDB) PutUser(user *types.User) error {

	credentialsJSON, err := json.Marshal(user.Credentials)
	if err != nil {
		return fmt.Errorf("error marshalling credentials: %s", err.Error())
	}

	if user.ID == "" {
		fmt.Println("inserting user")
		_, err = udb.db.Insert("curry_club.users").
			Rows(
				goqu.Record{
					"username":    user.Username,
					"credentials": credentialsJSON,
				},
			).Returning("id").Executor().ScanVal(&user.ID)
		if err != nil {
			return fmt.Errorf("error inserting user: %s", err.Error())
		}

		return nil
	}

	fmt.Println("updating user")
	res, err := udb.db.Update("curry_club.users").
		Where(goqu.Ex{"id": user.ID}).
		Set(
			goqu.Record{
				"username":    user.Username,
				"credentials": credentialsJSON,
			},
		).Executor().Exec()
	if err != nil {
		return fmt.Errorf("error updating user: %s", err.Error())
	}
	rowsAf, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %s", err.Error())
	}
	if rowsAf == 0 {
		return fmt.Errorf("no rows updated")
	}

	return nil
}
