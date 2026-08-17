package comap

import (
	"fmt"
	"time"

	"github.com/simonvetter/modbus"
 
	"towercore/internal/core/interfaces"
)

// Garante em tempo de compilação que TCPClient satisfaz o port.
var _ interfaces.ModbusClient = (*TCPClient)(nil)

// sentinelInactive é o valor devolvido pelo controlador ComAp para registos
// existentes mas não configurados/ativos (confirmado empiricamente no
// varrimento de descoberta: 0x8000 / 32768). Não deve ser interpretado como
// valor real de telemetria.
const sentinelInactive uint16 = 0x8000

// TCPClient implementa ModbusClient usando github.com/simonvetter/modbus,
// o mesmo driver validado no script de descoberta de campo.
type TCPClient struct {
	client *modbus.ModbusClient
}

// NewTCPClient abre uma ligação Modbus TCP a host:port e define o slave ID.
func NewTCPClient(host string, port int, slaveID uint8, timeout time.Duration) (*TCPClient, error) {
	endpoint := fmt.Sprintf("tcp://%s:%d", host, port)
	c, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     endpoint,
		Timeout: timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("comap: config modbus client: %w", err)
	}
	if err := c.Open(); err != nil {
		return nil, fmt.Errorf("comap: abrir ligação a %s: %w", endpoint, err)
	}
	c.SetUnitId(slaveID)
	return &TCPClient{client: c}, nil
}

// Close fecha a ligação Modbus TCP.
func (t *TCPClient) Close() error {
	return t.client.Close()
}

// ReadHoldingRegisters lê `quantity` registos holding a partir de `address`.
// Nesta primeira versão suporta apenas quantity=1 (alinhado com o padrão de
// leitura registo-a-registo usado no script de descoberta); ver TODO para
// leitura em bloco.
func (t *TCPClient) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	if quantity != 1 {
		return nil, fmt.Errorf("comap: leitura em bloco (quantity=%d) ainda não suportada", quantity)
	}

	val, err := t.client.ReadRegister(address, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, fmt.Errorf("registo %d: %w", address, err)
	}

	// 0x8000 = registo existe mas está inativo → não é falha de rede
	if val == sentinelInactive {
		return nil, ErrInactiveRegister
	}

	return []byte{byte(val >> 8), byte(val & 0xFF)}, nil
}