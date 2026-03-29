//go:build android

package main

import "C"

import (
	"encoding/json"
	"log"
	"net"
	"strings"
	"sync"

	"tailscale.com/net/netmon"
	"tailscale.com/net/netns"
)

var androidProtectNoopLog sync.Once
var androidInterfacesMu sync.RWMutex
var androidInterfaces []netmon.Interface

type androidInterfaceSnapshot struct {
	Name         string   `json:"name"`
	Index        int      `json:"index"`
	MTU          int      `json:"mtu"`
	Flags        int      `json:"flags"`
	HardwareAddr string   `json:"hardware_addr,omitempty"`
	Addrs        []string `json:"addrs,omitempty"`
}

func init() {
	netns.SetAndroidProtectFunc(func(fd int) error {
		androidProtectNoopLog.Do(func() {
			log.Printf("codexmobile: AndroidProtectFunc is running in no-op mode until full VPN/TUN setup is implemented")
		})
		_ = fd
		return nil
	})
	netmon.RegisterInterfaceGetter(func() ([]netmon.Interface, error) {
		androidInterfacesMu.RLock()
		defer androidInterfacesMu.RUnlock()
		return append([]netmon.Interface(nil), androidInterfaces...), nil
	})
}

//export CodexMobileSetAndroidDefaultRouteInterface
func CodexMobileSetAndroidDefaultRouteInterface(interfaceName *C.char) {
	if interfaceName == nil {
		netmon.UpdateLastKnownDefaultRouteInterface("")
		return
	}
	netmon.UpdateLastKnownDefaultRouteInterface(strings.TrimSpace(C.GoString(interfaceName)))
}

//export CodexMobileSetAndroidInterfaceSnapshot
func CodexMobileSetAndroidInterfaceSnapshot(snapshotJSON *C.char) {
	if snapshotJSON == nil {
		androidInterfacesMu.Lock()
		androidInterfaces = nil
		androidInterfacesMu.Unlock()
		return
	}

	var snapshots []androidInterfaceSnapshot
	if err := json.Unmarshal([]byte(C.GoString(snapshotJSON)), &snapshots); err != nil {
		log.Printf("codexmobile: failed to parse Android interface snapshot: %v", err)
		return
	}

	next := make([]netmon.Interface, 0, len(snapshots))
	for _, snapshot := range snapshots {
		name := strings.TrimSpace(snapshot.Name)
		if name == "" {
			continue
		}

		iface := netmon.Interface{
			Interface: &net.Interface{
				Index: snapshot.Index,
				MTU:   snapshot.MTU,
				Name:  name,
				Flags: net.Flags(snapshot.Flags),
			},
		}
		if snapshot.HardwareAddr != "" {
			if hw, err := net.ParseMAC(snapshot.HardwareAddr); err == nil {
				iface.Interface.HardwareAddr = hw
			}
		}
		if len(snapshot.Addrs) > 0 {
			iface.AltAddrs = make([]net.Addr, 0, len(snapshot.Addrs))
			for _, raw := range snapshot.Addrs {
				raw = strings.TrimSpace(raw)
				if raw == "" {
					continue
				}
				if _, ipNet, err := net.ParseCIDR(raw); err == nil {
					iface.AltAddrs = append(iface.AltAddrs, ipNet)
					continue
				}
				if ip := net.ParseIP(raw); ip != nil {
					iface.AltAddrs = append(iface.AltAddrs, &net.IPAddr{IP: ip})
				}
			}
		}
		next = append(next, iface)
	}

	androidInterfacesMu.Lock()
	androidInterfaces = next
	androidInterfacesMu.Unlock()
}
