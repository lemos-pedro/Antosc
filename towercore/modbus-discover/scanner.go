package main

import "fmt"

type Scanner struct {
	cfg Config
}

func NewScanner(cfg Config) *Scanner {
	return &Scanner{
		cfg: cfg,
	}
}

func (s *Scanner) Run() error {

	fmt.Println("--------------------------------")
	fmt.Println("MODBUS DISCOVERY")
	fmt.Println("--------------------------------")

	client, handler, ctx, err := NewClient(s.cfg)
	if err != nil {
		return err
	}

	defer handler.Close()

	fmt.Println("Ligado ao equipamento.")

	data, err := (*client).ReadHoldingRegisters(
		ctx,
		0,
		1,
	)

	if err != nil {
		return err
	}

	fmt.Printf("Recebidos %d bytes\n", len(data))
	fmt.Printf("% X\n", data)

	return nil
}
