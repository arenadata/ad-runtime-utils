package exec

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type SocketProtocol string

const (
	TCP  SocketProtocol = "tcp"
	UDP  SocketProtocol = "udp"
	TCP6 SocketProtocol = "tcp6"
	UDP6 SocketProtocol = "udp6"
)

// SocketInfo represents information about a socket.
// Protocol represents the protocol of the socket (TCP or UDP).
// IP represents the IP address of the socket.
// Port represents the port number of the socket.
type SocketInfo struct {
	Protocol SocketProtocol
	IP       net.IP
	Port     int
}

type NetworkSocketStat struct {
	SL            string
	LocalAddress  string
	RemoteAddress string
	State         string
	Queue         string
	Timer         string
	Retransmits   string
	UID           string
	Timeout       string
	Inode         string
}

func getInodeForPid(pid int) (map[string]int, error) {
	// Get file descriptors for the pid.
	fdLink := fmt.Sprintf("/proc/%d/fd", pid)
	fdDir, err := os.Open(fdLink)
	if err != nil {
		return nil, err
	}
	fdEntries, err := fdDir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	if cerr := fdDir.Close(); cerr != nil {
		return nil, cerr
	}
	inodes := make(map[string]int)
	for _, fdEntry := range fdEntries {
		var fdPath string
		fdPath, err = os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, fdEntry))
		// File descriptor might be closed already, so skip it
		if err != nil {
			continue
		}
		// Check if file descriptor is a socket
		if strings.HasPrefix(fdPath, "socket:[") {
			inode := strings.TrimPrefix(strings.TrimSuffix(fdPath, "]"), "socket:[")
			inodes[inode] = pid
		}
	}
	return inodes, nil
}

func parseNetworkStat(statFilePath string) ([]NetworkSocketStat, error) {
	tcpStatFile, err := os.Open(statFilePath)
	if err != nil {
		return nil, err
	}
	defer tcpStatFile.Close()

	var stats []NetworkSocketStat
	scanner := bufio.NewScanner(tcpStatFile)
	// Skip the header and return if there are no entries
	if !scanner.Scan() {
		return stats, nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		// Skip invalid fields
		tcpStatFieldsCount := 10
		if len(fields) < tcpStatFieldsCount {
			continue
		}
		stats = append(stats, NetworkSocketStat{
			SL:            fields[0],
			LocalAddress:  fields[1],
			RemoteAddress: fields[2],
			State:         fields[3],
			Queue:         fields[4],
			Timer:         fields[5],
			Retransmits:   fields[6],
			UID:           fields[7],
			Timeout:       fields[8],
			Inode:         fields[9],
		})
	}
	return stats, scanner.Err()
}

func getSocketsForInodes(inodes map[string]int, protocol SocketProtocol) ([]SocketInfo, error) {
	var statFilePath string
	switch protocol {
	case TCP:
		statFilePath = "/proc/net/tcp"
	case UDP:
		statFilePath = "/proc/net/udp"
	case TCP6:
		statFilePath = "/proc/net/tcp6"
	case UDP6:
		statFilePath = "/proc/net/udp6"
	default:
		return nil, fmt.Errorf("invalid socket protocol: %q", protocol)
	}

	netStats, err := parseNetworkStat(statFilePath)
	if err != nil {
		return nil, err
	}

	var sockets []SocketInfo
	for _, stat := range netStats {
		// Hex IP: addressParts[0], Hex Port AddressPart[1]
		addressFieldsCount := 2
		addressParts := strings.Split(stat.LocalAddress, ":")
		if len(addressParts) != addressFieldsCount {
			continue
		}

		var ipBytes []byte
		ipBytes, err = hex.DecodeString(addressParts[0])
		if err != nil {
			continue
		}
		// Convert hex IP to the readable format
		if len(ipBytes) != 4 && len(ipBytes) != 16 {
			continue
		}
		ip := net.IP(ipBytes)
		// Convert hex port to int
		var port int64
		port, err = strconv.ParseInt(addressParts[1], 16, 32)
		if err != nil {
			continue
		}
		// Check if the socket is in our inodes list, inode is fields[9]
		if _, exists := inodes[stat.Inode]; exists {
			sockets = append(sockets, SocketInfo{
				Protocol: protocol,
				IP:       ip,
				Port:     int(port),
			})
		}
	}
	return sockets, nil
}

// GetTCPSocketsForPid returns a list of Listening TCP sockets for the given process ID.
func GetTCPSocketsForPid(pid int) ([]SocketInfo, error) {
	inodes, err := getInodeForPid(pid)
	if err != nil {
		return nil, err
	}
	sockets, err := getSocketsForInodes(inodes, TCP)
	if err != nil {
		return nil, err
	}
	sockets6, err := getSocketsForInodes(inodes, TCP6)
	if err != nil {
		return nil, err
	}
	return append(sockets, sockets6...), nil
}

// GetUDPSocketsForPid returns a list of Listening UDP sockets for the given process ID.
func GetUDPSocketsForPid(pid int) ([]SocketInfo, error) {
	inodes, err := getInodeForPid(pid)
	if err != nil {
		return nil, err
	}
	sockets, err := getSocketsForInodes(inodes, UDP)
	if err != nil {
		return nil, err
	}
	sockets6, err := getSocketsForInodes(inodes, UDP6)
	if err != nil {
		return nil, err
	}
	return append(sockets, sockets6...), nil
}

// GetSocketsForPid returns a list of all Listening sockets for the given process ID.
func GetSocketsForPid(pid int) ([]SocketInfo, error) {
	tcpSockets, err := GetTCPSocketsForPid(pid)
	if err != nil {
		return nil, err
	}
	udpSockets, err := GetUDPSocketsForPid(pid)
	if err != nil {
		return nil, err
	}
	return append(tcpSockets, udpSockets...), nil
}
