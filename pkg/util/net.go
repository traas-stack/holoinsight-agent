/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"net"
	"os"
	"time"
)

var localIp string
var hostname string
var timezone string

func init() {
	{
		// TODO 多网卡场景
		// eth0
		firstIp := ""
		eth0Ip := ""

		interfaces, err := net.Interfaces()
		if err != nil {
			panic(err)
		}
	interfaceLoop:
		for _, i := range interfaces {
			if i.Name == "lo" {
				continue
			}
			if i.Flags&net.FlagUp != net.FlagUp {
				continue
			}
			if i.Flags&net.FlagLoopback == net.FlagLoopback {
				continue
			}
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				ipv4 := ipNet.IP.To4()
				if ipv4 == nil {
					continue
				}
				ip := ipv4.String()
				if firstIp == "" {
					firstIp = ip
				}
				if i.Name == "eth0" {
					eth0Ip = ip
					break interfaceLoop
				}
			}
		}

		if eth0Ip != "" {
			localIp = eth0Ip
		} else {
			localIp = firstIp
		}
	}

	{
		if h, err := os.Hostname(); err == nil && h != "" {
			hostname = h
		} else {
			hostname = localIp
		}
	}
}

// GetLocalIp get local ipv4
func GetLocalIp() string {
	return localIp
}

// GetHostname
func GetHostname() string {
	return hostname
}

func SetLocalIp(ip string) {
	localIp = ip
}

func GetLocalTimezone() string {
	tz, _ := time.Now().Zone()
	return tz
}

type (
	closeWrite interface {
		CloseWrite() error
	}
	closeRead interface {
		CloseRead() error
	}
)

func MaybeCloseWrite(conn interface{}) error {
	if x, ok := conn.(closeWrite); ok {
		return x.CloseWrite()
	}
	return nil
}

func MaybeCloseRead(conn interface{}) error {
	if x, ok := conn.(closeRead); ok {
		return x.CloseRead()
	}
	return nil
}
