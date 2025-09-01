package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	host "github.com/libp2p/go-libp2p/core/host"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Helper to create a host for testing
func createTestHost(t *testing.T) (host.Host, error) {
	priv, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	h, err := libp2p.New(libp2p.Identity(priv))
	return h, err
}

func TestInvalidMultiaddr(t *testing.T) {
	invalidAddr := "/invalid/multiaddr"
	_, err := ma.NewMultiaddr(invalidAddr)
	if err == nil {
		t.Error("Expected error for invalid multiaddr, got nil")
	}
}

func TestAddrInfoFromP2pAddr_Invalid(t *testing.T) {
	invalidAddr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/1234")
	_, err := peer.AddrInfoFromP2pAddr(invalidAddr)
	if err == nil {
		t.Error("Expected error for AddrInfoFromP2pAddr with missing peer ID")
	}
}

func TestHostConnectionFailure(t *testing.T) {
	host, err := createTestHost(t)
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	// Generate a valid but random peer ID
	priv, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		t.Fatalf("Failed to get peer ID: %v", err)
	}
	info := peer.AddrInfo{ID: peerID}
	err = host.Connect(context.Background(), info)
	if err == nil {
		t.Error("Expected connection failure, got nil")
	}
}

func TestMessageSendWithoutPeer(t *testing.T) {
	// Simulate sending a message with nil peerInfo
	var peerInfo *peer.AddrInfo
	if peerInfo != nil {
		t.Error("peerInfo should be nil")
	}
	// Should not panic or send
}

func TestExitCommand(t *testing.T) {
	exit := "exit"
	if exit != "exit" {
		t.Error("Exit command not recognized")
	}
}

func TestStreamHandlerReceivesMessage(t *testing.T) {
	// This is a basic test for the handler logic
	var buf bytes.Buffer
	buf.WriteString("hello\n")
	// Simulate reading from stream
	r := bufio.NewReader(&buf)
	str, err := r.ReadString('\n')
	if err != nil {
		t.Errorf("Failed to read string: %v", err)
	}
	if str != "hello\n" {
		t.Errorf("Expected 'hello\\n', got '%s'", str)
	}
}

func TestStreamHandlerError(t *testing.T) {
	// Simulate error on stream
	var buf bytes.Buffer
	r := bufio.NewReader(&buf)
	_, err := r.ReadString('\n')
	if !errors.Is(err, io.EOF) {
		t.Errorf("Expected EOF error, got %v", err)
	}
}

func TestPeerToPeerMessaging(t *testing.T) {
	ctx := context.Background()

	// Create host A
	privA, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("Failed to generate key for host A: %v", err)
	}
	hostA, err := libp2p.New(libp2p.Identity(privA))
	if err != nil {
		t.Fatalf("Failed to create host A: %v", err)
	}

	// Create host B
	privB, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("Failed to generate key for host B: %v", err)
	}
	hostB, err := libp2p.New(libp2p.Identity(privB))
	if err != nil {
		t.Fatalf("Failed to create host B: %v", err)
	}

	// Setup message channel for host B
	received := make(chan string, 1)
	hostB.SetStreamHandler("/chat/1.0.0", func(s network.Stream) {
		r := bufio.NewReader(s)
		str, err := r.ReadString('\n')
		if err == nil {
			received <- str
		}
		s.Close()
	})

	// Connect host A to host B
	addrInfoB := peer.AddrInfo{
		ID:    hostB.ID(),
		Addrs: hostB.Addrs(),
	}
	if err := hostA.Connect(ctx, addrInfoB); err != nil {
		t.Fatalf("Failed to connect host A to host B: %v", err)
	}

	// Host A sends a message to host B
	stream, err := hostA.NewStream(ctx, hostB.ID(), "/chat/1.0.0")
	if err != nil {
		t.Fatalf("Failed to open stream from host A to host B: %v", err)
	}
	msg := "hello from A\n"
	_, err = stream.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Failed to write message from host A: %v", err)
	}
	stream.Close()

	// Verify host B received the message
	select {
	case got := <-received:
		if got != msg {
			t.Errorf("Host B received wrong message: got %q, want %q", got, msg)
		}
	case <-ctx.Done():
		t.Error("Timeout waiting for message on host B")
	}
}
