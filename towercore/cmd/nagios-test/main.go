// cmd/nagios-test/main.go (descartável, só para validar)
package main

import (
	"context"
	"fmt"
	"time"

	"towercore/internal/adapters/nagios"
)

func main() {
	client := nagios.NewClient("http://172.17.0.31", "nagiosadmin", "123", 10*time.Second)
	status, err := client.FetchHostStatus(context.Background(), "A102")
	if err != nil {
		fmt.Println("erro:", err)
		return
	}
	fmt.Printf("%+v\n", status)
}