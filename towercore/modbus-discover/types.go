package main

type Register struct {
	Address uint16
	Value   uint16
}

type ScanResult struct {
	Registers []Register
}