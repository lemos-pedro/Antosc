package main

import (
	"fmt"
	"log"
	"os"

	"towercore/internal/adapters/neteco"
)

func main() {
	baseURL := "https://192.168.9.11:31943"
	username := "A.lemos"
	password := os.Getenv("NETECO_PASSWORD")
	if password == "" {
		log.Fatal("defina NETECO_PASSWORD")
	}

	client := neteco.NewClient(baseURL, username, password)
	if err := client.Login(); err != nil {
		log.Fatal("login falhou: ", err)
	}
	fmt.Println("login OK")

	// siteDn de teste — substituir pelos 4 confirmados
	sites := map[string]string{
		"KNROD001_Praceta":     "NE=33554471",
		"HBHUB001_Macolocolo":  "NE=33554510",
		"LDVIA003_SAOJOSE":     "NE=33554467",
		"LDVIA004_VILA FLOR":   "NE=33554498",
	}

	for name, dn := range sites {
		status, err := client.FetchBatteryStatus(dn)
		if err != nil {
			fmt.Printf("%s: erro %v\n", name, err)
			continue
		}
		fmt.Printf("%s: %+v\n", name, status)
	}
}