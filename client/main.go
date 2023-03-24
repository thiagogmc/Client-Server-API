package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Quotation struct {
	Bid string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:8080/cotacao", nil)
	if err != nil {
		fmt.Println("It was not possible to create the new request with context. Err: ", err)
		return
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("It was not possible to make the request. Err: ", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Println("It was not possible to complete the request. The server answered with timeout status code: ", res.StatusCode)
		return
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("It was not possible to read the response body. Err: ", err)
		return
	}

	var quotation Quotation
	err = json.Unmarshal(body, &quotation)
	if err != nil {
		fmt.Println("It was not possible to unmarshalling json. Err: ", err)
		return
	}
	f, err := os.OpenFile("cotacao.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("It was not possible to create the file. Err: ", err)
		return
	}
	defer f.Close()
	_, err = f.WriteString("Dólar: " + quotation.Bid + "\n")
	if err != nil {
		fmt.Println("It was not possible to write on file. Err: ", err)
		return
	}
}
