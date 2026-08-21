package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	data, err := os.ReadFile("docs/apifox/MeteorX-backend.apifox.json")
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Println("JSON INVALID:", err)
		os.Exit(1)
	}
	fmt.Println("JSON VALID")
}