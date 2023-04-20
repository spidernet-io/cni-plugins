package ipchecking

import (
	"context"
	"fmt"
	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
	"net"
	"net/netip"
	"time"
)

func IPCheckingByARP(ifi *net.Interface, targetIP netip.Addr, retry int, interval time.Duration) error {
	client, err := arp.Dial(ifi)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var conflictingMac string
	// start a goroutine to receive arp response
	go func() {
		var packet *arp.Packet
		for {
			select {
			case <-ctx.Done():
				return
			default:
				packet, _, err = client.Read()
				if err != nil {
					cancel()
					return
				}

				if packet.Operation == arp.OperationReply {
					// found reply and simple check if the reply packet is we want.
					if packet.SenderIP.Compare(targetIP) == 0 {
						conflictingMac = packet.SenderHardwareAddr.String()
						cancel()
						return
					}
				}
			}
		}
	}()

	// we send a gratuitous arp to checking if ip is conflict
	// we use dad mode(duplicate address detection mode), so
	// we set source ip to 0.0.0.0
	packet, err := arp.NewPacket(arp.OperationRequest, ifi.HardwareAddr, netip.MustParseAddr("0.0.0.0"), ethernet.Broadcast, targetIP)
	if err != nil {
		cancel()
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	stop := false
	for i := 0; i < retry && !stop; i++ {
		select {
		case <-ctx.Done():
			stop = true
		case <-ticker.C:
			err = client.WriteTo(packet, ethernet.Broadcast)
			if err != nil {
				stop = true
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to checking ip %s if it's conflicting: %v", targetIP.String(), err)
	}

	if conflictingMac != "" {
		// found ip conflicting
		return fmt.Errorf("pod's interface %s with an conflicting ip %s, %s is located at %s", ifi.Name,
			targetIP.String(), targetIP.String(), conflictingMac)
	}

	return nil
}
