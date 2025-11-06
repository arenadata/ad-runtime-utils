package exec

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/arenadata/ad-runtime-utils/internal/config"
)

const (
	PortHealthCheckType              = "port"
	PortHealthCheckTimeoutParamName  = "timeout"
	PortHealthCheckTimeoutDefault    = 60
	PortHealthCheckPortParamName     = "port"
	PortHealthCheckProtocolParamName = "protocol"
	PortHealthCheckProtocolDefault   = TCP
)

type HealthCheck interface {
	Check() error
}

// PortHealthCheck checks that a given port is open by the process with a given PID.
type PortHealthCheck struct {
	Port     int
	Timeout  int
	PID      int
	Config   config.HealthCheckConfig
	Protocol SocketProtocol
}

func (h *PortHealthCheck) Check() error {
	var infos []SocketInfo
	var err error

	if err = h.parseConfig(); err != nil {
		return err
	}

	endTime := time.Now().Add(time.Duration(h.Timeout) * time.Second)
	for time.Now().Before(endTime) {
		switch h.Protocol {
		case TCP, TCP6:
			if infos, err = GetTCPSocketsForPid(h.PID); err != nil {
				fmt.Fprintf(os.Stderr, "Error getting TCP sockets for PID, retrying: %v\n", err)
			}
		case UDP, UDP6:
			if infos, err = GetUDPSocketsForPid(h.PID); err != nil {
				fmt.Fprintf(os.Stderr, "Error getting UDP sockets for PID, retrying: %v\n", err)
			}
		default:
			return fmt.Errorf("invalid socket protocol: %s", h.Protocol)
		}
		for _, info := range infos {
			if info.Port == h.Port {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("port %d not open after %d seconds", h.Port, h.Timeout)
}

func (h *PortHealthCheck) parseConfig() error {
	// Port
	_, found := h.Config.Params[PortHealthCheckPortParamName]
	if !found {
		return fmt.Errorf("missing %s parameter", PortHealthCheckPortParamName)
	}
	portStr, ok := h.Config.Params[PortHealthCheckPortParamName].(string)
	if !ok {
		return fmt.Errorf("parameter %s has invalid value", PortHealthCheckPortParamName)
	}
	var port int
	var err error
	port, err = strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("parameter %s has invalid value: %w", PortHealthCheckPortParamName, err)
	}
	h.Port = port

	// Protocol
	_, found = h.Config.Params[PortHealthCheckProtocolParamName]
	if !found {
		h.Protocol = PortHealthCheckProtocolDefault
	} else {
		var protocol string
		protocol, ok = h.Config.Params[PortHealthCheckProtocolParamName].(string)
		if !ok {
			return fmt.Errorf("parameter %s has invalid value", PortHealthCheckProtocolParamName)
		}
		h.Protocol = SocketProtocol(protocol)
	}

	// Timeout
	_, ok = h.Config.Params[PortHealthCheckTimeoutParamName].(string)
	if !ok {
		h.Timeout = PortHealthCheckTimeoutDefault
	} else {
		var timeoutStr string
		timeoutStr, ok = h.Config.Params[PortHealthCheckTimeoutParamName].(string)
		if !ok {
			return fmt.Errorf("parameter %s has invalid value", PortHealthCheckTimeoutParamName)
		}
		var timeout int
		timeout, err = strconv.Atoi(timeoutStr)
		if err != nil {
			return fmt.Errorf("parameter %s has invalid value: %w", PortHealthCheckTimeoutParamName, err)
		}
		h.Timeout = timeout
	}
	return nil
}
