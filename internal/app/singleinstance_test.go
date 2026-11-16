package app

import (
	"net"
	"slices"
	"testing"
	"time"
)

// The ports are the app's and the user's own, in range, and the same from
// one launch to the next.
func TestInstancePorts(t *testing.T) {
	ports := instancePorts("MyGo 云音乐")
	if len(ports) != 8 {
		t.Fatalf("%d ports", len(ports))
	}
	if !slices.Equal(ports, instancePorts("MyGo 云音乐")) {
		t.Error("another launch took other ports")
	}
	if slices.Equal(ports, instancePorts("MyGo Music")) {
		t.Error("two apps meet on the same ports")
	}
	for _, p := range ports {
		if p < 25000 || p >= 32000 {
			t.Errorf("port %d out of range", p)
		}
	}
}

// A launch asks the instance running to show itself, and only answers of
// this app count.
func TestInstanceHandshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	shown := make(chan struct{}, 1)
	serveInstance(ln, "MyGo 云音乐", func() { shown <- struct{}{} })

	addr := ln.Addr().String()
	if !askInstance(addr, "MyGo 云音乐") {
		t.Fatal("no answer")
	}
	select {
	case <-shown:
	case <-time.After(time.Second):
		t.Fatal("asked, but did not show")
	}
	if askInstance(addr, "MyGo Music") {
		t.Error("another app answered")
	}
}
