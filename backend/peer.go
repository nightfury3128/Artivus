package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func handleStream(s network.Stream) {
	fmt.Println("📩 Incoming stream opened!")
	r := bufio.NewReader(s)
	for {
		str, err := r.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Stream closed")
			return
		}
		fmt.Printf("💬 Received: %s", str)
	}
}

func main() {
	ctx := context.Background()

	// --- Generate identity (use persistent keypair in future) ---
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}

	// --- Create a new libp2p host ---
	host, err := libp2p.New(
		libp2p.Identity(priv),
	)
	if err != nil {
		panic(err)
	}

	// --- Setup stream handler ---
	host.SetStreamHandler("/chat/1.0.0", handleStream)

	fmt.Println("✅ Peer started!")
	fmt.Println("Peer ID:", host.ID())
	for _, addr := range host.Addrs() {
		fmt.Printf("➡️ Share this multiaddr: %s/p2p/%s\n", addr, host.ID())
	}

	// --- Prompt for peer to connect to ---
	fmt.Print("Enter target peer full multiaddr (leave empty to wait): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	targetAddr := scanner.Text()

	var peerInfo *peer.AddrInfo
	if targetAddr != "" {
		maddr, err := ma.NewMultiaddr(targetAddr)
		if err != nil {
			fmt.Println("❌ Invalid multiaddr:", err)
			return
		}
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			fmt.Println("❌ Failed to parse peer info:", err)
			return
		}
		peerInfo = info

		// --- Connect to peer ---
		if err := host.Connect(ctx, *info); err != nil {
			fmt.Println("❌ Connection failed:", err)
			return
		}
		fmt.Println("✅ Connected to peer:", info.ID)
	}

	// --- Chat loop ---
	for {
		fmt.Print("✏️ Enter message (or 'exit'): ")
		scanner.Scan()
		msg := scanner.Text()
		if strings.TrimSpace(msg) == "exit" {
			break
		}
		if peerInfo != nil {
			s, err := host.NewStream(ctx, peerInfo.ID, "/chat/1.0.0")
			if err != nil {
				fmt.Println("❌ Failed to open stream:", err)
				continue
			}
			_, err = s.Write([]byte(msg + "\n"))
			if err != nil {
				fmt.Println("❌ Failed to send:", err)
			}
			s.Close()
		} else {
			fmt.Println("⚠️ No peer connected.")
		}
	}

	fmt.Println("👋 Exiting...")
	select {}
}
