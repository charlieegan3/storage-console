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

func (udb *UserDB) GetUser(name string) (*types.User, error) {

	var user struct {
		ID          uint64 `db:"id"`
		Name        string `db:"name"`
		Credentials []byte `db:"credentials"`
	}
	_, err := udb.db.From("curry_club.users").
		Where(goqu.Ex{"name": name}).
		ScanStruct(&user)
	if err != nil {
		return nil, fmt.Errorf("error getting user '%s': %s", name, err.Error())
	}

	var credentials []webauthn.Credential
	err = json.Unmarshal(user.Credentials, &credentials)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling credentials: %s", err.Error())
	}

	return &types.User{
		ID:          user.ID,
		Name:        user.Name,
		Credentials: credentials,
	}, nil
}

func (udb *UserDB) GetUserByID(id uint64) (*types.User, error) {

	var user struct {
		ID          uint64 `db:"id"`
		Name        string `db:"name"`
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
		Name:        user.Name,
		Credentials: credentials,
	}, nil
}

func (udb *UserDB) AddCredentialsForUser(user *types.User, credentials []webauthn.Credential) error {

	if user.ID == 0 {
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

	if user.ID == 0 {
		fmt.Println("inserting user")
		_, err = udb.db.Insert("curry_club.users").
			Rows(
				goqu.Record{
					"name":        user.Name,
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
				"name":        user.Name,
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
