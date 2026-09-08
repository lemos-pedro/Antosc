package interfaces

import (
	"context"
	"time"
)

type HostState string

const (
	HostStatePending     HostState = "pending"
	HostStateUp          HostState = "up"
	HostStateDown        HostState = "down"
	HostStateUnreachable HostState = "unreachable"
	HostStateUnknown     HostState = "unknown"
)

type HostStatus struct {
	Hostname             string
	State                HostState
	PluginOutput         string
	LastCheck            time.Time
	LastStateChange      time.Time
	NotificationsEnabled bool
	ChecksEnabled        bool
}

// NagiosClient é o port — core/services só depende disto, nunca de
// detalhes HTTP/CGI do adapter concreto.
type NagiosClient interface {
	FetchHostStatus(ctx context.Context, hostname string) (HostStatus, error)
}
