package interfaces

import "errors"

// ErrTowerEndpointNotFound segue o mesmo padrão de ErrTowerNotFound /
// ErrEventNotFound / ErrTicketNotFound já existentes no pacote.
var ErrTowerEndpointNotFound = errors.New("tower endpoint not found")
