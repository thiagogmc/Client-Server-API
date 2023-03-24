package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Quotation struct {
	Code       string  `json:"code"`
	Codein     string  `json:"codein"`
	Name       string  `json:"name"`
	High       float64 `json:"high,string"`
	Low        float64 `json:"low,string"`
	VarBid     float64 `json:"varBid,string"`
	PctChange  string  `json:"pctChange"`
	Bid        float64 `json:"bid,string"`
	Ask        string  `json:"ask"`
	Timestamp  string  `json:"timestamp"`
	CreateDate string  `json:"create_date"`
}

type App struct {
	DB *sql.DB
}

func main() {
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	app := App{
		DB: db,
	}
	err = app.prepareDatabase()
	if err != nil {
		fmt.Println("It was not possible to create database tables")
		return
	}

	http.HandleFunc("/cotacao", app.quotationHandler)
	http.ListenAndServe(":8080", nil)
}

func (app *App) quotationHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancel()

	quotation, err := app.getQuotation(ctx)
	if err != nil {
		fmt.Println("It was not possible to get the quotation. Err: ", err)
		http.Error(w, "Request timeout", http.StatusRequestTimeout)
		return
	}

	ctxDB, cancelDB := context.WithTimeout(r.Context(), 10*time.Millisecond)
	defer cancelDB()
	app.insertQuotation(ctxDB, quotation)
	if err != nil {
		fmt.Println("It was not possible to insert the quotation on database. Error:", err)
		http.Error(w, "Request timeout", http.StatusRequestTimeout)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	decoder := json.NewEncoder(w)
	decoder.Encode(quotation)
}

func skipRoot(jsonBlob []byte) json.RawMessage {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(jsonBlob, &root); err != nil {
		panic(err)
	}
	for _, v := range root {
		return v
	}
	return nil
}

func (app *App) prepareDatabase() error {
	_, err := app.DB.Exec(`CREATE TABLE IF NOT EXISTS quotations (
		code TEXT,
		codein TEXT,
		name TEXT,
		high REAL,
		low REAL,
		varBid REAL,
		pctChange TEXT,
		bid REAL,
		ask TEXT,
		timestamp TEXT,
		create_date TEXT
	)`)
	if err != nil {
		return err
	}
	return nil
}

func (app *App) insertQuotation(ctx context.Context, q Quotation) error {
	stmt, err := app.DB.Prepare(`INSERT INTO quotations (code, codein, name, high, low, varBid, pctChange, bid, ask, timestamp, create_date)
                        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, q.Code, q.Codein, q.Name, q.High, q.Low, q.VarBid, q.PctChange, q.Bid, q.Ask, q.Timestamp, q.CreateDate)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return err
	default:
		return nil
	}
}

func (app *App) getQuotation(ctx context.Context) (Quotation, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return Quotation{}, err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return Quotation{}, err
	}
	defer response.Body.Close()

	select {
	case <-ctx.Done():
		return Quotation{}, err
	default:
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return Quotation{}, err
		}
		var quotation Quotation
		err = json.Unmarshal(skipRoot(body), &quotation)
		if err != nil {
			return Quotation{}, err
		}
		return quotation, nil
	}
}
