package nodes

import (
	"errors"
	"net"
	"os/exec"
	"sync"
	"time"

	vultr "github.com/JamesClonk/vultr/lib"
	"github.com/astaxie/beego"
)

var master string

type Node struct {
	Users    int
	IsMaster bool
	vultr.Server
}

var buffer int

// the magic parameter to adjust
const Multiple = 50

// first node is master
var node_mux sync.Mutex
var nodes []Node
var cand_mux sync.Mutex
var cand_nodes []*Node
var busy_nodes []*Node

// the first non local ipv4 address
func GetMaster() error {
	master = ""
	ifaces, err := net.Interfaces()
	// handle err
	if err != nil {
		beego.Error(err)
		return err
	}
LABEL:
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		// handle err
		if err != nil {
			beego.Error(err)
			return err
		}
	NEXT:
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// skip ipv6
			if ip.To4() == nil {
				continue NEXT
			}

			// process IP address
			if ip.Equal(net.ParseIP("127.0.0.1")) {
				continue LABEL
			} else {
				master = ip.String()
				return nil
			}
		}
	}
	return errors.New("can't work with 42")
}

func deleteNode(node *Node) error {
	if node.IsMaster {
		beego.Trace("trying to delete master")
		return nil
	}
	client := vultr.NewClient(beego.AppConfig.String("key"), nil)
	err := client.DeleteServer(node.Server.ID)
	node_mux.Lock()
	for i, n := range nodes {
		// remove
		if node.Server.ID == n.Server.ID {
			nodes = append(nodes[:i], nodes[i+1:]...)
			break
		}
	}
	node_mux.Unlock()
	return err
}

func checkStat(node *Node) {
	if node.IsMaster {
		return
	}
	times := 0
AGAIN:
	time.Sleep(time.Duration(10*times) * time.Second)
	times += 1
	done := make(chan error, 1)
	cmd := exec.Command("bash", "-c", "ssh root@"+node.Server.MainIP+
		" -p 26 ls /root/done")
	err := cmd.Start()
	if err != nil {
		beego.Trace(err)
		if times <= 8 {
			goto AGAIN
		} else {
			goto DESTROY
		}
	}
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(10 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			beego.Error("failed to kill: ", err)
		}
		beego.Trace("process killed as timeout reached")
		if times <= 8 {
			goto AGAIN
		} else {
			goto DESTROY
		}
	case err := <-done:
		if err != nil {
			beego.Trace(err)
			if times <= 8 {
				goto AGAIN
			} else {
				goto DESTROY
			}
		} else {
			beego.Trace(node.Server.MainIP + " is UP")
			cand_mux.Lock()
			cand_nodes = append([]*Node{node}, cand_nodes...)
			buffer += Multiple
			cand_mux.Unlock()
			return
		}
	}

DESTROY:
	beego.Trace(node.Server.MainIP + " will be destroyed")
	deleteNode(node)
	if err != nil {
		beego.Error(node.Server.ID + " not destroyed")
	}
}

// retrieve nodes
func RetrieveNodes() error {
	client := vultr.NewClient(beego.AppConfig.String("key"), nil)
	servers, err := client.GetServers()
	if err != nil {
		beego.Error(err)
		return err
	}

	for _, serv := range servers {
		if serv.MainIP == master {
			// prepend to nodes, master is the first node
			nodes = append([]Node{Node{0, true, serv}}, nodes...)
			// the master must be the cand
			cand_nodes = append([]*Node{&nodes[0]}, cand_nodes...)
			buffer = Multiple / 2
		} else {
			// append to nodes
			nodes = append(nodes, Node{0, false, serv})
		}
	}

	if len(cand_nodes) == 0 {
		beego.Trace("no master in list")
		return errors.New("no master in list")
	}

	for _, node := range nodes[1:] {
		go checkStat(&node)
	}

	return nil
}
