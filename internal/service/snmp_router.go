package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

type InterfaceBandwidth struct {
	Name   string `json:"name"`
	InBps  int64  `json:"in_bps"`
	OutBps int64  `json:"out_bps"`
}

func (s *SNMPService) GetRouterInterfaceBandwidth(ctx context.Context, ip string, community string, interfaceNames []string) ([]InterfaceBandwidth, error) {
	if community == "" {
		community = "public"
	}

	client := s.newSNMPClient(ip, community)
	if err := client.Connect(); err != nil {
		return []InterfaceBandwidth{}, fmt.Errorf("connect snmp: %w", err)
	}
	defer client.Conn.Close()

	sample1, err := s.pollInterfaceCounters(client, interfaceNames)
	if err != nil {
		return []InterfaceBandwidth{}, err
	}

	time.Sleep(1 * time.Second)

	sample2, err := s.pollInterfaceCounters(client, interfaceNames)
	if err != nil {
		return []InterfaceBandwidth{}, err
	}

	results := make([]InterfaceBandwidth, 0)
	for _, name := range interfaceNames {
		s2, ok2 := sample2[name]
		s1, ok1 := sample1[name]
		if ok1 && ok2 {
			inBps := int64(s2.In - s1.In) * 8
			outBps := int64(s2.Out - s1.Out) * 8
			results = append(results, InterfaceBandwidth{
				Name:   name,
				InBps:  inBps,
				OutBps: outBps,
			})
		}
	}

	return results, nil
}

type counters struct {
	In  uint64
	Out uint64
}

func (s *SNMPService) pollInterfaceCounters(client *gosnmp.GoSNMP, interfaceNames []string) (map[string]counters, error) {
	descResults, err := client.BulkWalkAll(".1.3.6.1.2.1.2.2.1.2")
	if err != nil {
		return nil, err
	}

	nameToIdx := make(map[string]int)
	for _, v := range descResults {
		name := cleanSNMPString(snmpString(v))
		idx := extractIfIndex(v.Name, ".1.3.6.1.2.1.2.2.1.2")
		nameToIdx[name] = idx
	}

	results := make(map[string]counters)
	for _, name := range interfaceNames {
		idx, ok := nameToIdx[name]
		if !ok {
			continue
		}

		oids := []string{
			fmt.Sprintf(".1.3.6.1.2.1.31.1.1.1.6.%d", idx),
			fmt.Sprintf(".1.3.6.1.2.1.31.1.1.1.10.%d", idx),
		}
		resp, err := client.Get(oids)
		if err != nil {
			continue
		}

		var c counters
		for _, v := range resp.Variables {
			if strings.HasPrefix(v.Name, ".1.3.6.1.2.1.31.1.1.1.6") {
				if val, ok := v.Value.(uint64); ok {
					c.In = val
				}
			} else if strings.HasPrefix(v.Name, ".1.3.6.1.2.1.31.1.1.1.10") {
				if val, ok := v.Value.(uint64); ok {
					c.Out = val
				}
			}
		}
		results[name] = c
	}

	return results, nil
}

func (s *SNMPService) GetRouterInterfaces(ctx context.Context, ip string, community string) ([]MonitorInterface, error) {
	if community == "" {
		community = "public"
	}

	client := s.newSNMPClient(ip, community)
	if err := client.Connect(); err != nil {
		return []MonitorInterface{}, fmt.Errorf("connect snmp: %w", err)
	}
	defer client.Conn.Close()

	ifaces, err := s.pollInterfaces(client)
	if err != nil {
		return []MonitorInterface{}, err
	}
	return ifaces, nil
}
