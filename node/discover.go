/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/p2p-go/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

func Subscribe(ctx context.Context, peernode *core.PeerNode, l *logger.Lg, minerRecord MinerRecord) {
	var (
		err      error
		room     string
		findpeer peer.AddrInfo
	)
	fmt.Printf("Subscribe logger point: %p\n", l)
	fmt.Printf("Subscribe peernode point: %p\n", peernode)
	fmt.Printf("Subscribe minerRecord point: %p\n", &minerRecord)
	fmt.Println("start discover...")
	l.Discover("info", "start discover...")

	gossipSub, err := pubsub.NewGossipSub(ctx, peernode.GetHost())
	if err != nil {
		l.Discover("err", fmt.Sprintf("NewGossipSub: %v", err))
		return
	}
	bootnode := peernode.GetBootnode()
	if strings.Contains(bootnode, "12D3KooWRm2sQg65y2ZgCUksLsjWmKbBtZ4HRRsGLxbN76XTtC8T") {
		room = fmt.Sprintf("%s-12D3KooWRm2sQg65y2ZgCUksLsjWmKbBtZ4HRRsGLxbN76XTtC8T", core.NetworkRoom)
	} else if strings.Contains(bootnode, "12D3KooWEGeAp1MvvUrBYQtb31FE1LPg7aHsd1LtTXn6cerZTBBd") {
		room = fmt.Sprintf("%s-12D3KooWEGeAp1MvvUrBYQtb31FE1LPg7aHsd1LtTXn6cerZTBBd", core.NetworkRoom)
	} else if strings.Contains(bootnode, "12D3KooWGDk9JJ5F6UPNuutEKSbHrTXnF5eSn3zKaR27amgU6o9S") {
		room = fmt.Sprintf("%s-12D3KooWGDk9JJ5F6UPNuutEKSbHrTXnF5eSn3zKaR27amgU6o9S", core.NetworkRoom)
	} else if strings.Contains(bootnode, "12D3KooWS8a18xoBzwkmUsgGBctNo6QCr6XCpUDR946mTBBUTe83") {
		room = fmt.Sprintf("%s-12D3KooWS8a18xoBzwkmUsgGBctNo6QCr6XCpUDR946mTBBUTe83", core.NetworkRoom)
	} else if strings.Contains(bootnode, "12D3KooWDWeiiqbpNGAqA5QbDTdKgTtwX8LCShWkTpcyxpRf2jA9") {
		room = fmt.Sprintf("%s-12D3KooWDWeiiqbpNGAqA5QbDTdKgTtwX8LCShWkTpcyxpRf2jA9", core.NetworkRoom)
	} else if strings.Contains(bootnode, "12D3KooWNcTWWuUWKhjTVDF1xZ38yCoHXoF4aDjnbjsNpeVwj33U") {
		room = fmt.Sprintf("%s-12D3KooWNcTWWuUWKhjTVDF1xZ38yCoHXoF4aDjnbjsNpeVwj33U", core.NetworkRoom)
	} else {
		room = core.NetworkRoom
	}

	l.Discover("info", fmt.Sprintf("room: %s", room))

	// setup local mDNS discovery
	if err := setupDiscovery(peernode.GetHost()); err != nil {
		l.Discover("err", fmt.Sprintf("setupDiscovery: %v", err))
		return
	}

	// join the pubsub topic called librum
	topic, err := gossipSub.Join(room)
	if err != nil {
		l.Discover("err", fmt.Sprintf("Join: %v", err))
		return
	}

	// subscribe to topic
	subscriber, err := topic.Subscribe()
	if err != nil {
		l.Discover("err", fmt.Sprintf("Subscribe: %v", err))
		return
	}

	l.Discover("info", fmt.Sprintf("Join room: %s", room))
	fmt.Println("Join room: ", room)
	for {
		msg, err := subscriber.Next(ctx)
		if err != nil {
			l.Discover("err", fmt.Sprintf("subscriber.Next: %v", err))
			continue
		}

		// only consider messages delivered by other peers
		if msg.ReceivedFrom == peernode.GetHost().ID() {
			continue
		}

		err = json.Unmarshal(msg.Data, &findpeer)
		if err != nil {
			l.Discover("err", fmt.Sprintf("Unmarshal: %v", err))
			continue
		}
		fmt.Println("got peer: ", findpeer.ID.String())
		l.Discover("info", fmt.Sprintf("got peer: %s", findpeer.ID.String()))
		minerRecord.SavePeer(findpeer)
	}
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID.String())
	err := n.h.Connect(context.TODO(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.String(), err)
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, "", &discoveryNotifee{h: h})
	return s.Start()
}
