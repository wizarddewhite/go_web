package nodes

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
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

const Multiple = 50

var cand_nodes []Node
var busy_nodes []Node

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

func checkStat(node *Node) {
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
			cand_nodes = append([]Node{*node}, cand_nodes...)
			return
		}
	}

DESTROY:
	beego.Trace(node.Server.MainIP + " will be destroyed")
}

// retrieve nodes
func RetrieveNodes() error {
	client := vultr.NewClient(beego.AppConfig.String("key"), nil)
	servers, err := client.GetServers()
	if err != nil {
		beego.Error(err)
		return err
	}

	nodes := make([]Node, 0, 10)
	for _, serv := range servers {
		if serv.MainIP == master {
			cand_nodes = append([]Node{Node{0, true, serv}}, cand_nodes...)
			buffer = Multiple / 2
		} else {
			nodes = append([]Node{Node{0, false, serv}}, nodes...)
		}
	}

	if len(cand_nodes) == 0 {
		beego.Trace("no master in list")
		return errors.New("no master in list")
	}

	for _, node := range nodes {
		go checkStat(&node)
	}

	fmt.Println(len(cand_nodes))
	fmt.Println(len(busy_nodes))

	// remove
	//cand_nodes = append(cand_nodes[:0], cand_nodes[1:]...)
	return nil
}
