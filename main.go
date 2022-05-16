package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/netip"
	"os"
	"sync"

	"github.com/grilix/toonels/internal"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	yaml "gopkg.in/yaml.v2"
)

type TunnelEndpoints struct {
	Local  string `yaml:"local"`
	Target string `yaml:"target"`
}

type JumpNode struct {
	User            string            `yaml:"user"`
	PrivateKeyPath  string            `yaml:"private_key_path"`
	ServerPublicKey string            `yaml:"server_public_key"`
	Addr            string            `yaml:"addr"`
	Tunnels         []TunnelEndpoints `yaml:"tunnels"`
}

type TunnelsFile struct {
	Jump []JumpNode `yaml:"nodes"`
}

func PrivateKeyFile(file string) (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}

func NewSSHNode(jump JumpNode) (*internal.SSHNode, error) {
	// TODO:
	// serverPublicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(jump.ServerPublicKey))
	// if err != nil {
	//     return nil, errors.Wrap(
	//         err,
	//         fmt.Sprintf("can't parse server public key %s", jump.ServerPublicKey),
	//     )
	// }
	auth, err := PrivateKeyFile(jump.PrivateKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(
			"can't load private key at %s",
			jump.PrivateKeyPath,
		))
	}
	config := &ssh.ClientConfig{
		User: jump.User,
		Auth: []ssh.AuthMethod{auth},
		// TODO:
		// HostKeyCallback: ssh.FixedHostKey(key),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	jumpAddr, err := netip.ParseAddrPort(jump.Addr)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(
			"can't parse jump address %s",
			jump.Addr,
		))
	}

	return &internal.SSHNode{
		Config: config,
		Addr:   jumpAddr,
	}, nil
}

func NewTunnel(jump *internal.SSHNode, local, target string) (*internal.SSHTunnel, error) {
	localEndpoint, err := netip.ParseAddrPort(local)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(
			"can't parse local address %s",
			local,
		))
	}
	targetEndpoint, err := netip.ParseAddrPort(target)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(
			"can't parse target address %s",
			target,
		))
	}

	return &internal.SSHTunnel{
		Jump:   jump,
		Local:  localEndpoint,
		Target: targetEndpoint,
	}, nil
}

func main() {
	var tunnelsFile TunnelsFile
	yamlFile, err := ioutil.ReadFile(".tunnels.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlFile, &tunnelsFile)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds)
	var wg sync.WaitGroup

	wg.Add(len(tunnelsFile.Jump))

	for _, jump := range tunnelsFile.Jump {
		jumpNode, err := NewSSHNode(jump)
		if err != nil {
			log.Fatal(err)
		}

		for _, endpoints := range jump.Tunnels {
			tunnel, err := NewTunnel(jumpNode, endpoints.Local, endpoints.Target)
			if err != nil {
				log.Fatal(err)
			}
			tunnel.Log = logger

			go func(tunnel *internal.SSHTunnel) {
				log.Fatal(tunnel.Start())
			}(tunnel)
		}
	}

	wg.Wait()
}
