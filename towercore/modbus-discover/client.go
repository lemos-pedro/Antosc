package main

import (
	"context"
	"fmt"
	"time"

	mb "github.com/grid-x/modbus"
)

func NewClient(cfg Config) (*mb.Client, *mb.TCPClientHandler, context.Context, error) {

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	handler := mb.NewTCPClientHandler(addr)

	handler.Timeout = 5 * time.Second
	handler.SlaveID = byte(cfg.Slave)

	ctx := context.Background()

	if err := handler.Connect(ctx); err != nil {
		return nil, nil, nil, err
	}

	client := mb.NewClient(handler)

	return &client, handler, ctx, nil
}