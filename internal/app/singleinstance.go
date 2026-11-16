package app

import (
	"bufio"
	"hash/fnv"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// instanceMagic opens what a launch says to the instance running, and what
// that instance answers; the app's name follows it, so that another app
// that happens to hold the port is not taken for this one.
const instanceMagic = "mygo-cloudmusic/instance"

// instancePorts are the loopback ports this app's instances meet on, in
// the order they try them: derived from the app's name and the user's, so
// that another user, mygo dev and the installed app do not answer each
// other. They sit below the ranges the systems give connection ports.
func instancePorts(name string) []int {
	h := fnv.New32a()
	_, _ = io.WriteString(h, name+"\x00"+userName())
	base := 25000 + int(h.Sum32()%7000)
	ports := make([]int, 8)
	for i := range ports {
		ports[i] = base + i
	}
	return ports
}

// userName names the session the app runs in, for instancePorts.
func userName() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return os.Getenv("USERNAME")
}

// instanceServer is an instance listening for a later launch of the app.
type instanceServer struct {
	ln net.Listener
}

// serveInstances starts listening for a later launch of the app, which
// asks the instance running to show its window; show runs on that
// request. It reports the instance already running instead, if one
// answers, and neither when every port is taken.
func serveInstances(name string, show func()) (srv *instanceServer, running bool) {
	for _, port := range instancePorts(name) {
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		// Two launches at once both find the port free: the one that
		// loses the bind asks again, letting the winner answer.
		for range 5 {
			if askInstance(addr, name) {
				return nil, true
			}
			ln, err := net.Listen("tcp", addr)
			if err == nil {
				serveInstance(ln, name, show)
				return &instanceServer{ln: ln}, false
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil, false
}

// serveInstance answers launches on ln, showing the window when one asks.
func serveInstance(ln net.Listener, name string, show func()) {
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
				line, err := bufio.NewReader(conn).ReadString('\n')
				if err != nil || !strings.HasPrefix(line, instanceMagic+" "+name+" ") {
					return
				}
				show()
				_, _ = io.WriteString(conn, instanceMagic+" "+name+" ok\n")
			}(conn)
		}
	}()
}

// askInstance asks the instance on addr to show itself, and reports
// whether this app's instance answered.
func askInstance(addr, name string) bool {
	allowForeground()
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, instanceMagic+" "+name+" show\n"); err != nil {
		return false
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	return err == nil && strings.HasPrefix(line, instanceMagic+" "+name+" ")
}
