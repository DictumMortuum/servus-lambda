package main

import (
	"database/sql"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
)

var (
	db_uri = os.Getenv("POSTGRESQL_ADDON_URI")
)

type Wish struct {
	Id         int    `json:"Id" db:"id"`
	CalendarId int    `json:"CalendarId" db:"calendar_id"`
	Owner      string `json:"Owner" db:"owner"`
	Status     string `json:"Status" db:"status"`
	Desc       string `json:"Desc" db:"description"`
	Title      string `json:"Title" db:"title"`
}

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)

func router(req events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	switch req.RequestContext.HTTP.Method {
	case "GET":
		return get(req)
	case "POST":
		return upsert(req)
	default:
		return clientError(http.StatusMethodNotAllowed)
	}
}

func get(req events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	db, err := sqlx.Connect("postgres", db_uri)
	if err != nil {
		return serverError(err)
	}
	defer db.Close()

	var rs []Wish

	err = db.Select(&rs, `select * from twishes`)
	if err != nil {
		return serverError(err)
	}

	js, err := json.Marshal(rs)
	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(js),
	}, nil
}

func Exists(db *sqlx.DB, data Wish) (*sql.NullInt64, error) {
	var id sql.NullInt64

	stmt, err := db.PrepareNamed("select id from twishes where id = :id")
	if err != nil {
		return nil, err
	}

	err = stmt.Get(&id, data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &id, nil
}

func upsert(req events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	db, err := sqlx.Connect("postgres", db_uri)
	if err != nil {
		return serverError(err)
	}
	defer db.Close()

	var rs Wish

	err = json.Unmarshal([]byte(req.Body), &rs)
	if err != nil {
		return serverError(err)
	}

	id, err := Exists(db, rs)
	if err != nil {
		return serverError(err)
	}

	if id == nil {
		_, err = db.NamedExec(`
		insert into twishes (
			id,
			calendar_id,
			owner,
			status,
			description,
			title
		) values (
			:id,
			:calendar_id,
			:owner,
			:status,
			:description,
			:title
		)
	`, &rs)
		if err != nil {
			return serverError(err)
		}
	} else {
		_, err = db.NamedExec(`
			update twishes set
				status = :status
			where
				id = :id and
				calendar_id = :calendar_id and
				owner = :owner
		`, &rs)
		if err != nil {
			return serverError(err)
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(req.Body),
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	errorLogger.Println(err.Error())

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}

func main() {
	lambda.Start(router)
}
