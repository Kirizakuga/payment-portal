package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	orderCode := 1234567
	jsonBytes, _ := json.Marshal(map[string]interface{}{"orderCode": orderCode})
	
	var jsonObj map[string]interface{}
	json.Unmarshal(jsonBytes, &jsonObj)
	
	value := jsonObj["orderCode"]
	fmt.Printf("Value type: %T\n", value)
	fmt.Printf("Value with %%v: %v\n", value)
}
