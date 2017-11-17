package nodes

import (
	"errors"
	"fmt"
	"net"

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

// retrieve nodes
func RetrieveNodes() error {
	client := vultr.NewClient(beego.AppConfig.String("key"), nil)
	servers, err := client.GetServers()
	if err != nil {
		beego.Error(err)
		return err
	}

	hasmaster := false
	for _, serv := range servers {
		if serv.MainIP == master {
			hasmaster = true
			cand_nodes = append([]Node{Node{0, true, serv}}, cand_nodes...)
		} else {
			cand_nodes = append([]Node{Node{0, false, serv}}, cand_nodes...)
		}
	}

	if !hasmaster {
		beego.Trace("no master in list")
		return errors.New("no master in list")
	}

	for _, node := range cand_nodes {
		beego.Trace("%s: %s\n", node.Server.ID, node.IsMaster)
	}

	fmt.Println(len(cand_nodes))
	fmt.Println(len(busy_nodes))

	// remove
	//cand_nodes = append(cand_nodes[:0], cand_nodes[1:]...)
	return nil
}
