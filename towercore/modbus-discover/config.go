package main

type Config struct {
	Host  string
	Port  int
	Slave int

	Start uint
	End   uint

	Block uint

	Type string
}