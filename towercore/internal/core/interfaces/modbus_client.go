package interfaces

// ModbusClient é o port (contrato) para leitura de registos holding Modbus.
// Implementações concretas (TCP, RTU) vivem em internal/adapters/comap.
// core/ nunca importa o driver Modbus diretamente — apenas esta interface.
type ModbusClient interface {
	// ReadHoldingRegisters lê `quantity` registos holding a partir de
	// `address` e devolve os bytes em big-endian. Deve devolver erro
	// explícito para registos inativos/desconfigurados (ex.: sentinel
	// 0x8000 no ComAp), nunca um valor fabricado.
	ReadHoldingRegisters(address, quantity uint16) ([]byte, error)

	// Close liberta a ligação subjacente.
	Close() error
}