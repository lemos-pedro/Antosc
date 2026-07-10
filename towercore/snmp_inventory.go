package main

import (
	"context"
	"fmt"
	"time"

	mb "github.com/grid-x/modbus"
)

var slaveIDs = []byte{
	0,
	1,
	2,
	3,
	4,
	5,
	10,
	16,
	100,
	200,
	247,
	248,
	249,
	250,
	251,
	252,
	253,
	254,
	255,
}

func main() {

	host := "192.168.149.10"
	port := 502

	addr := fmt.Sprintf("%s:%d", host, port)

	fmt.Println("========================================")
	fmt.Println(" MODBUS DISCOVERY")
	fmt.Println("========================================")
	fmt.Println()

	for _, slave := range slaveIDs {

		fmt.Printf("Slave %-3d ", slave)

		handler := mb.NewTCPClientHandler(addr)
		handler.Timeout = 3 * time.Second
		handler.SlaveID = slave

		ctx := context.Background()

		err := handler.Connect(ctx)
		if err != nil {
			fmt.Println("-> ligação falhou")
			continue
		}

		client := mb.NewClient(handler)

		holdingOK := false
		inputOK := false

		_, err = client.ReadHoldingRegisters(ctx, 0, 1)
		if err == nil {
			holdingOK = true
		}

		_, err = client.ReadInputRegisters(ctx, 0, 1)
		if err == nil {
			inputOK = true
		}

		handler.Close()

		if !holdingOK && !inputOK {
			fmt.Println("-> sem resposta Modbus")
			continue
		}

		fmt.Printf("-> ")

		if holdingOK {
			fmt.Print("FC03 ")
		}

		if inputOK {
			fmt.Print("FC04 ")
		}

		fmt.Println("OK")
	}

	fmt.Println()
	fmt.Println("Discovery terminado.")
}