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
	Limit    int
	IsMaster bool
	IsOut    bool
	IsCand   bool
	vultr.Server
}

var buff_mux sync.Mutex
var buffer int

// the magic parameter to adjust
const N = 3
const Multiple = 50

// first node is master
var node_mux sync.Mutex
var nodes []Node
var cand_mux sync.Mutex
var cand_nodes []*Node
var busy_nodes []*Node

var index int
var cu chan int
var cleanup_cond *sync.Cond

func init() {
	index = 0

	cu = make(chan int)
	cleanup_cond = sync.NewCond(new(sync.Mutex))
	go cleanup_nodes()
}

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
	var err error
	client := vultr.NewClient(beego.AppConfig.String("key"), nil)
	for range [30]struct{}{} {
		err = client.DeleteServer(node.Server.ID)
		if err == nil || err.Error() != "Unable to destroy server: Servers cannot be destroyed within 5 minutes of being created" {
			break
		}
		time.Sleep(10 * time.Second)
	}
	beego.Info(node.Server.MainIP + " is deleted")
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
		if times <= 5 {
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
		if times <= 5 {
			goto AGAIN
		}
	case err := <-done:
		if err != nil {
			beego.Trace(err)
			if times <= 5 {
				goto AGAIN
			}
		} else {
			beego.Info(node.Server.MainIP + " is ready")
			node.Users = -1 // abuse Users to indicate setup done
			cand_mux.Lock()
			node.IsCand = true // mark it added to cand_nodes
			cand_nodes = append([]*Node{node}, cand_nodes...)
			buffer += node.Limit
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
		isout := false

		if serv.ServerState == "ok" &&
			serv.CurrentBandwidth >= (serv.AllowedBandwidth*0.9) {
			isout = true
		}

		if serv.MainIP == master {
			// prepend to nodes, master is the first node
			nodes = append([]Node{Node{0, Multiple / 2, true, isout, true, serv}}, nodes...)
			// the master must be the cand
			cand_nodes = append([]*Node{&nodes[0]}, cand_nodes...)
			buffer = Multiple / 2
		} else {
			// append to nodes
			nodes = append(nodes, Node{0, Multiple, false, isout, false, serv})
		}
	}

	if len(cand_nodes) == 0 {
		beego.Trace("no master in list")
		return errors.New("no master in list")
	}

	for i, _ := range nodes[1:] {
		go checkStat(&nodes[i+1])
	}

	return nil
}

// create a node
func CreateNode() {
	var server vultr.Server
	var err error
	client := vultr.NewClient(beego.AppConfig.String("key"), nil)
	// Amsterdam, Frankfurt, Paris
	regions := [...]int{7, 9, 24}
	for _, r := range regions {
		server, err = client.CreateServer("test", r, 201, 241, nil)
		if err == nil {
			break
		} else if err.Error() != "Plan is not available in the selected datacenter.  This could mean you have chosen the wrong plan (for example, a storage plan in a location that does not offer them), or the location you have selected does not have any more capacity." {
			beego.Error(err)
			return
		}
	}

	beego.Info("Trying to create: ", server.ID)
	// wait for installation, until state is ok
	time.Sleep(2 * time.Minute)
	for range [20]struct{}{} {
		server, err = client.GetServer(server.ID)
		if server.ServerState == "ok" {
			break
		}
		time.Sleep(20 * time.Second)
	}

	// dup_machine
	time.Sleep(30 * time.Second)

	done := make(chan error, 1)
	cmd := exec.Command("bash", "-c", "/root/dup_machine/dup_machine.sh "+server.MainIP+" \""+server.DefaultPassword+"\"")
	err = cmd.Start()
	if err != nil {
		beego.Trace(err)
	}
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(5 * time.Minute):
		if err := cmd.Process.Kill(); err != nil {
			beego.Error("failed to kill: ", err)
		}
		beego.Trace("process killed as timeout reached")
	case err := <-done:
		if err != nil {
			beego.Trace(err)
		} else {
			beego.Trace(server.MainIP + " dup_machine executed")
		}
	}

	// Check the setup state
	node := new(Node)
	node.Users = 0
	node.IsMaster = false
	node.Limit = Multiple
	node.Server = server
	checkStat(node)

	// node is up, add to nodes
	if node.Users == -1 {
		node.Users = 0
		node_mux.Lock()
		nodes = append(nodes, *node)
		node_mux.Unlock()
	}
}

func GetServiceNode() string {
	var ip string
	if len(cand_nodes) != 0 {
		cand_mux.Lock()
		ip = cand_nodes[0].Server.MainIP
		cand_mux.Unlock()
	} else {
		beego.Warn("running out of nodes")
		ip = master
	}
	return ip
}

func GetNode(ip string) *Node {
	node_mux.Lock()
	for i, n := range nodes {
		if n.Server.MainIP == ip {
			node_mux.Unlock()
			return &nodes[i]
		}
	}
	node_mux.Unlock()
	return nil
}

func UpdateBuffer(delta int) {
	buff_mux.Lock()
	buffer -= delta
	buff_mux.Unlock()
	beego.Trace("current buffer is ", buffer)
}

func CheckNodeBandwidth(n *Node) error {
	// the node is already removed from cand_nodes
	if n.IsOut {
		return nil
	}

	client := vultr.NewClient(beego.AppConfig.String("key"), nil)
	server, err := client.GetServer(n.Server.ID)

	if err != nil {
		return err
	}

	// running out of bandwidth remove it from cand_node
	out := server.CurrentBandwidth >= (server.AllowedBandwidth * 0.9)
	if out && n.IsMaster {
		beego.Warn("Master is full")
	}

	if out {
		cand_mux.Lock()
		for i, c := range cand_nodes {
			if c.Server.ID == n.Server.ID {
				cand_nodes = append(cand_nodes[:i], cand_nodes[i+1:]...)
				n.IsOut = out
				n.IsCand = false
				buffer -= n.Limit - n.Users
				// delete the node after there is no connection
				break
			}
		}
		cand_mux.Unlock()
	}
	return nil
}

func CheckNodeUsers(n *Node) {
	if !n.IsCand {
		return
	}

	if n.Users >= n.Limit {
		cand_mux.Lock()
		for i, c := range cand_nodes {
			if c.Server.ID == n.Server.ID {
				cand_nodes = append(cand_nodes[:i], cand_nodes[i+1:]...)
				n.IsCand = false
				break
			}
		}
		cand_mux.Unlock()
	}
}

func cleanup_nodes() {
	cleanup_cond.L.Lock()
	cleanup_cond.Wait()
	var full_nodes []*Node
	node_mux.Lock()
	for i, n := range nodes {
		// if node is out of bandwidth and no user, delete it
		if !n.IsMaster && n.IsOut && (n.Users == 0) {
			full_nodes = append(full_nodes, &n)
		}
		// if node still has bandwidth and has 1/5 Multiply space
		// add it to cand_nodes
		if !n.IsCand && !n.IsOut && (n.Users < (n.Limit * 4 / 5)) {
			cand_mux.Lock()
			if n.IsMaster {
				// add it to the last if this is master node
				cand_nodes = append(cand_nodes, &nodes[i])
			} else {
				// or add it to the second last if not
				cand_nodes = append(cand_nodes[:len(cand_nodes)-1],
					&nodes[i], cand_nodes[len(cand_nodes)-1])
			}
			nodes[i].IsCand = true
			buffer += n.Limit - n.Users
			cand_mux.Unlock()
		}

	}
	node_mux.Unlock()

	// How we manage the number of nodes:
	// 1. We always keep N number of alive nodes
	// 2. buffer is lower than (Multiple / 2), create a Node
	// 3. buffer is higher than (2 * Multiple), delete a none user one
	if buffer < (Multiple / 2) {
		go CreateNode()
	} else if (len(nodes)-len(full_nodes)) > N && buffer > (2*Multiple) {

		cand_mux.Lock()
		for i, c := range cand_nodes[1:] {
			if c.Users == 0 && !c.IsMaster {
				cand_nodes = append(cand_nodes[:i+1], cand_nodes[i+2:]...)
				buffer -= Multiple - c.Users
				go deleteNode(c)
				break
			}
		}
		cand_mux.Unlock()
	}

	// delete Node if necessary
	for _, n := range full_nodes {
		go deleteNode(n)
	}

	cleanup_cond.L.Unlock()

	index = 0
	go cleanup_nodes()
}

func Cleanup() {

	if index == 0 {
		go func() {
			select {
			case <-cu:
				cleanup_cond.L.Lock()
				cleanup_cond.Signal()
				cleanup_cond.L.Unlock()
			case <-time.After(60 * time.Second):
				cleanup_cond.L.Lock()
				cleanup_cond.Signal()
				cleanup_cond.L.Unlock()
			}
		}()
	}

	index += 1

	if index == len(nodes) {
		cu <- index
	}
}
