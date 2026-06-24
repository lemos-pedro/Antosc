package snmp

// EnterpriseVendorResolver implementa services.VendorResolver usando o
// mapa de enterprise OIDs definido em vendor_mapper.go. É um wrapper
// trivial: existe só para o core/services poder depender de uma
// interface, em vez de chamar uma função de adapters/snmp diretamente
// (regra do projeto: core nunca depende de adapters).
type EnterpriseVendorResolver struct{}

func NewEnterpriseVendorResolver() *EnterpriseVendorResolver {
	return &EnterpriseVendorResolver{}
}

func (r *EnterpriseVendorResolver) VendorFromSysObjectID(sysObjectID string) string {
	return VendorFromSysObjectID(sysObjectID)
}
