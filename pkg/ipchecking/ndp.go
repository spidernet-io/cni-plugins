package ipchecking

import (
	"errors"
	"fmt"
	"github.com/mdlayher/ndp"
	"github.com/spidernet-io/cni-plugins/pkg/constant"
	"net"
	"net/netip"

	"time"
)

var errRetry = errors.New("retry")

func IPCheckingByNDP(ifi *net.Interface, target netip.Addr, retry int, interval time.Duration) error {
	client, _, err := ndp.Listen(ifi, ndp.LinkLocal)
	if err != nil {
		return err
	}
	defer client.Close()

	m := &ndp.NeighborSolicitation{
		TargetAddress: target,
		Options: []ndp.Option{
			&ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      ifi.HardwareAddr,
			},
		},
	}

	var replyMac string
	replyMac, err = sendReceiveLoop(retry, interval, client, m, target)
	switch err {
	case constant.NDPFoundReply:
		if replyMac != ifi.HardwareAddr.String() {
			return fmt.Errorf("pod's interface %s with an conflicting ip %s, %s is located at %s", ifi.Name,
				target.String(), target.String(), replyMac)
		}
	case constant.NDPRetryError:
		return constant.NDPRetryError
	default:
		return fmt.Errorf("failed to checking ip conflicting: %v", err)
	}

	return nil
}

func sendReceiveLoop(retry int, interval time.Duration, client *ndp.Conn, msg ndp.Message, dst netip.Addr) (string, error) {
	var hwAddr string
	var err error
	for i := 0; i < retry; i++ {
		hwAddr, err = sendReceive(client, msg, dst, interval)
		switch err {
		case errRetry:
			continue
		case nil:
			return hwAddr, constant.NDPFoundReply
		default:
			return "", err
		}
	}

	return "", constant.NDPRetryError
}

func sendReceive(client *ndp.Conn, m ndp.Message, target netip.Addr, interval time.Duration) (string, error) {
	// Always multicast the message to the target's solicited-node multicast
	// group as if we have no knowledge of its MAC address.
	snm, err := ndp.SolicitedNodeMulticast(target)
	if err != nil {
		return "", fmt.Errorf("failed to determine solicited-node multicast address: %v", err)
	}

	// we send a gratuitous neighbor solicitation to checking if ip is conflict
	err = client.WriteTo(m, nil, snm)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}

	if err := client.SetReadDeadline(time.Now().Add(interval)); err != nil {
		return "", fmt.Errorf("failed to set deadline: %v", err)
	}

	msg, _, _, err := client.ReadFrom()
	if err == nil {
		na, ok := msg.(*ndp.NeighborAdvertisement)
		if ok && na.TargetAddress.Compare(target) == 0 && len(na.Options) == 1 {
			// found ndp reply what we want
			option, ok := na.Options[0].(*ndp.LinkLayerAddress)
			if ok {
				return option.Addr.String(), nil
			}
		}
		return "", errRetry

	}

	// Was the error caused by a read timeout, and should the loop continue?
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		return "", errRetry
	}

	return "", fmt.Errorf("failed to read message: %v", err)
}
