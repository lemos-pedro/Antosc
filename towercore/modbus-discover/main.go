package main

import (
	"fmt"
	"log"
	"time"

	"github.com/simonvetter/modbus"
)

func main() {
	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:           "tcp+tls://192.168.198.5:502",
		Timeout:       5 * time.Second,
		TLSClientCert: nil,
		TLSRootCAs:    nil,
	})
	if err != nil {
		log.Fatal(err)
	}

	client.SetUnitId(1)

	if err := client.Open(); err != nil {
		log.Fatal("erro ao abrir ligação TLS: ", err)
	}
	defer client.Close()

	for addr := uint16(0); addr < 200; addr += 20 {
		regs, err := client.ReadRegisters(addr, 20, modbus.HOLDING_REGISTER)
		if err != nil {
			fmt.Printf("addr %d: erro %v\n", addr, err)
			continue
		}
		fmt.Printf("addr %d: %v\n", addr, regs)
	}
}