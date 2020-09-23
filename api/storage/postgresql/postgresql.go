package postgresql

import (
	"database/sql"
	"log"

	"github.com/jonfriesen/subscriber-tracker-api/model"

	_ "github.com/lib/pq"
)

// PostgreSQL is our DB object that satifies our interface
type PostgreSQL struct {
	db *sql.DB
}

// NewConnection creates a DB connection
func NewConnection(databaseURL string) (*PostgreSQL, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	sqlStatement := `
	CREATE TABLE IF NOT EXISTS subscribers (
		id SERIAL PRIMARY KEY,
		name TEXT,
		email TEXT UNIQUE NOT NULL
	  );`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		return nil, err
	}

	return &PostgreSQL{
		db: db,
	}, nil
}

// ListSubscribers subs
func (p *PostgreSQL) ListSubscribers() ([]*model.Subscriber, error) {
	rows, err := p.db.Query("SELECT name, email FROM subscribers ORDER BY id DESC LIMIT 50;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subs := []*model.Subscriber{}
	for rows.Next() {
		var name string
		var email string
		err = rows.Scan(&name, &email)
		if err != nil {
			return nil, err
		}
		log.Println(">>> Received User:", name)
		subs = append(subs, &model.Subscriber{
			Name:  name,
			Email: email,
		})
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return subs, nil
}

// AddSubscriber adds
func (p *PostgreSQL) AddSubscriber(newSub *model.Subscriber) (*model.Subscriber, error) {
	sqlStatement := `
	INSERT INTO subscribers (name, email)
	VALUES ($1, $2)`
	_, err := p.db.Exec(sqlStatement, newSub.Name, newSub.Email)
	if err != nil {
		return nil, err
	}

	return newSub, nil
}

func (p *PostgreSQL) Close() error {
	return p.db.Close()
}

func (p *PostgreSQL) IsUp() error {
	return p.db.Ping()
}
