package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type UsdBrl struct {
	USDBRL USDBRL
}

type USDBRL struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type Quotation struct {
	Id       string
	Currency string
	Value    float64
}

func NewQuotation(currency string, value float64) *Quotation {
	return &Quotation{
		Id:       uuid.NewString(),
		Currency: currency,
		Value:    value,
	}
}

func main() {
	http.HandleFunc("/cotacao", handler)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10000*time.Millisecond)
	defer cancel()

	quotation, err := GetDolarQuotation(ctx)
	if err != nil {
		fmt.Println("Request external API error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx = context.Background()
	ctx, cancel = context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()

	dolarQuotation := []byte(quotation.USDBRL.Bid)
	err = storeDatabase(ctx, dolarQuotation)
	if err != nil {
		fmt.Println("Database store error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(dolarQuotation)
}

func storeDatabase(ctx context.Context, dolarQuotation []byte) error {
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/goexpert")
	if err != nil {
		return err
	}
	defer db.Close()

	dolarPrice := float64(dolarQuotation[0]) // TODO: fix convertion.
	quotation := NewQuotation("Dolar", dolarPrice)

	stmt, err := db.Prepare("insert into quotations (id, currency, value) values (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, quotation.Id, quotation.Currency, quotation.Value)
	if err != nil {
		return err
	}
	return nil
}

func GetDolarQuotation(ctx context.Context) (*UsdBrl, error) {
	// Prepare a request to an API target.
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	// Do the request.
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSON to a struct.
	var result UsdBrl
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
