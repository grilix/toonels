package internal

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"
	"sync"

	"golang.org/x/crypto/ssh"
)

type SSHNode struct {
	Addr   netip.AddrPort
	Config *ssh.ClientConfig
}

type SSHTunnel struct {
	Jump   *SSHNode
	Local  netip.AddrPort
	Target netip.AddrPort
	Log    *log.Logger
}

func (tunnel *SSHTunnel) logErr(err error) {
	if tunnel.Log != nil {
		tunnel.Log.Printf("%s\n", err)
	}
}

func (tunnel *SSHTunnel) logf(f string, args ...any) {
	if tunnel.Log == nil {
		return
	}
	tunnel.Log.Output(2, fmt.Sprintf(f, args...))
}

func (tunnel *SSHTunnel) Start() error {
	tunnel.logf(
		"Starting: %s -> %s -> %s",
		tunnel.Local.String(),
		tunnel.Jump.Addr.String(),
		tunnel.Target.String(),
	)

	listener, err := net.Listen("tcp", tunnel.Local.String())
	if err != nil {
		return err
	}
	defer listener.Close()

	serverConn, err := ssh.Dial(
		"tcp",
		tunnel.Jump.Addr.String(),
		tunnel.Jump.Config,
	)
	if err != nil {
		return err
	}
	defer serverConn.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		tunnel.logf("accepted connection: %s", tunnel.Local.String())
		go tunnel.forward(serverConn, conn)
	}
}

func (tunnel *SSHTunnel) forward(serverConn *ssh.Client, localConn net.Conn) {
	remoteConn, err := serverConn.Dial("tcp", tunnel.Target.String())
	if err != nil {
		tunnel.logErr(err)
		return
	}

	tunnel.logf(
		"%s -> %s\n",
		tunnel.Jump.Addr.String(),
		tunnel.Target.String(),
	)
	copyConn := func(writer, reader net.Conn, name string) {
		count, err := io.Copy(writer, reader)
		if err != nil {
			tunnel.logf("io.Copy error: %s", err)
		}
		tunnel.logf("%d bytes from %s", count, name)
	}

	var wg sync.WaitGroup

	wg.Add(2)
	defer localConn.Close()
	defer remoteConn.Close()

	go func() {
		defer wg.Done()

		copyConn(localConn, remoteConn, tunnel.Local.String())
	}()

	go func() {
		defer wg.Done()

		copyConn(remoteConn, localConn, tunnel.Target.String())
	}()

	wg.Wait()
}
