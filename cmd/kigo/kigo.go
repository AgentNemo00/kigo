package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"bytes"

	"github.com/agentnemo00/kigo/module"
)

func main() {
	payload := &module.PayloadStartUp{
		QueuePosition: 1,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error occurred while marshaling the payload:", err)
		return
	}
	resp, err := http.Post("http://localhost:10001/v1/KigoTextModule/notification/startup", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error occurred while making the request:", err)
		return
	}
	data := make(map[string]any, 0)
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error occurred while decoding the response:", err)
		return
	}
	fmt.Println("Response from KigoTextModule:", data)
}
