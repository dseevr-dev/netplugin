/***
Copyright 2014 Cisco Systems Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/contiv/netplugin/core"
	"github.com/contiv/netplugin/crtclient"
	"github.com/samalba/dockerclient"
	"github.com/vishvananda/netlink"

	log "github.com/Sirupsen/logrus"
)

type DockerConfig struct {
	Docker struct {
		Socket string
	}
}

type Docker struct {
	Client *dockerclient.DockerClient
}

func (d *Docker) Init(config *crtclient.Config) error {
	if config == nil {
		return core.Errorf("null config!")
	}

	cfg, ok := config.V.(*DockerConfig)

	if !ok {
		return core.Errorf("Invalid config type passed!")
	}

	if cfg.Docker.Socket == "" {
		return core.Errorf("Invalid arguments. cfg: %v", config)
	}

	// TODO: ADD TLS support
	d.Client, _ = dockerclient.NewDockerClient(cfg.Docker.Socket, nil)

	return nil
}

func (d *Docker) Deinit() {
}

func (d *Docker) getContPid(ctx *crtclient.ContainerEPContext) (string, error) {

	contNameOrID := ctx.NewContName
	if ctx.NewAttachUUID != "" {
		contNameOrID = ctx.NewAttachUUID
	}

	contInfo, err := d.Client.InspectContainer(contNameOrID)
	if err != nil {

		log.Printf("unable to get container info for '%s' \n",
			contNameOrID)
		return "", core.Errorf("couldn't obtain container info")
	}

	// the hack below works only for running containers
	if !contInfo.State.Running {
		return "", core.Errorf("container not running")
	}

	return strconv.Itoa(contInfo.State.Pid), nil
}

func setIfNs(ifname string, pid int) error {
	link, err := netlink.LinkByName(ifname)
	if err != nil {
		if !strings.Contains(err.Error(), "Link not found") {
			log.Printf("unable to find link %q error %q", ifname, err)
			return err
		}
		// try once more as sometimes (somehow) link creation is taking
		// sometime, causing link not found error
		time.Sleep(1 * time.Second)
		link, err = netlink.LinkByName(ifname)
		if err != nil {
			log.Printf("unable to find link %q error %q", ifname, err)
			return err
		}
	}

	err = netlink.LinkSetNsPid(link, pid)
	if err != nil {
		log.Printf("unable to move interface '%s' to pid %d \n",
			ifname, pid)
	}

	return err
}

// Note: most of the work in this function is a temporary workaround for
// what docker daemon would eventually do; in the meanwhile
// the essense of the logic is borrowed from pipework
func (d *Docker) moveIfToContainer(ctx *crtclient.ContainerEPContext) error {

	// log.Printf("Moving interface '%s' into container '%s' \n", ifID, contName)

	contPid, err := d.getContPid(ctx)
	if err != nil {
		log.Printf("error '%s' querying container name %s, uuid %s\n",
			err, ctx.NewContName, ctx.NewAttachUUID)
		return err
	}

	netnsDir := "/var/run/netns"

	err = os.Mkdir(netnsDir, 0700)
	if err != nil && !os.IsExist(err) {
		log.Printf("error creating '%s' direcotry \n", netnsDir)
		return err
	}

	netnsPidFile := path.Join(netnsDir, contPid)
	err = os.Remove(netnsPidFile)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("error removing file '%s' \n", netnsPidFile)
		return err
	} else {
		err = nil
	}

	procNetNs := path.Join("/proc", contPid, "ns/net")
	err = os.Symlink(procNetNs, netnsPidFile)
	if err != nil {
		log.Printf("error symlink file '%s' with '%s' \n", netnsPidFile)
		return err
	}

	intPid, _ := strconv.Atoi(contPid)
	err = setIfNs(ctx.InterfaceID, intPid)
	if err != nil {
		log.Printf("err '%s' moving if '%s' into container '%s' namespace\n",
			err, ctx.InterfaceID, ctx.NewContName)
		return err
	}

	return err
}

func (d *Docker) cleanupNetns(ctx *crtclient.ContainerEPContext) error {

	contPid, err := d.getContPid(ctx)
	if err != nil {
		return err
	}

	netnsPidFile := path.Join("/var/run/netns", contPid)
	_, err = os.Stat(netnsPidFile)
	if err != nil && os.IsExist(err) {
		os.Remove(netnsPidFile)
	}

	return nil
}

/*
func (d *Docker) configureIfAddress(ctx *crtclient.ContainerEPContext) error {

	log.Printf("configuring ip: addr -%s/%d- on if %s for container %s\n",
		ctx.IPAddress, ctx.SubnetLen, ctx.InterfaceID, ctx.NewContName)

	if ctx.IPAddress == "" {
		return nil
	}
	if ctx.SubnetLen == 0 {
		core.Errorf("Subnet mask unspecified \n")
	}

	contPid, err := d.getContPid(ctx)
	if err != nil {
		return err
	}

	intPid, err := strconv.Atoi(contPid)
	if err != nil {
		return err
	}

	contNs, err := netns.GetFromPid(intPid)
	if err != nil {
		log.Printf("error '%s' getting namespace for pid %s \n",
			err, contPid)
		return err
	}
	defer contNs.Close()

	origNs, err := netns.Get()
	if err != nil {
		log.Printf("error '%s' getting orig namespace\n", err)
		return err
	}

	defer origNs.Close()

	err = netns.Set(contNs)
	if err != nil {
		log.Printf("error '%s' setting netns \n", err)
		return err
	}
	defer netns.Set(origNs)

	link, err := netlink.LinkByName(ctx.InterfaceID)
	if err != nil {
		log.Printf("error '%s' getting if '%s' information \n", err,
			ctx.InterfaceID)
		return err
	}

	addr, err := netlink.ParseAddr(ctx.IPAddress + "/" +
		strconv.Itoa((int)(ctx.SubnetLen)))
	if err != nil {
		log.Printf("error '%s' parsing ip %s/%d \n", err,
			ctx.IPAddress, ctx.SubnetLen)
		return err
	}

	err = netlink.AddrAdd(link, addr)
	if err != nil {
		log.Printf("## netlink add addr failed '%s' \n", err)
		return err
	}

	err = netlink.LinkSetUp(link)
	if err != nil {
		log.Printf("## netlink set link up failed '%s' \n", err)
		return err
	}

	return err
}
*/
func (d *Docker) configureIfAddress(ctx *crtclient.ContainerEPContext) error {

	log.Printf("configuring ip: addr -%s/%d- on if %s for container %s\n",
		ctx.IPAddress, ctx.SubnetLen, ctx.InterfaceID, ctx.NewContName)

	if ctx.IPAddress == "" {
		return nil
	}
	if ctx.SubnetLen == 0 {
		core.Errorf("Subnet mask unspecified \n")
	}

	contPid, err := d.getContPid(ctx)
	if err != nil {
		return err
	}

	out, err := exec.Command("/sbin/ip", "netns", "exec", contPid, "ip",
		"addr", "add", ctx.IPAddress+"/"+strconv.Itoa(int(ctx.SubnetLen)),
		"dev", ctx.InterfaceID).Output()
	if err != nil {
		log.Printf("error configuring ip address for interface %s "+
			"%s out = '%s', err = '%s'\n", ctx.InterfaceID, out, err)
		return err
	}

	out, err = exec.Command("/sbin/ip", "netns", "exec", contPid, "ip",
		"link", "set", ctx.InterfaceID, "up").Output()
	if err != nil {
		log.Printf("error bringing interface %s up 'out = %s', err = %s\n",
			ctx.InterfaceID, out, err)
		return err
	}
	log.Printf("successfully configured ip and brought up the interface \n")

	return err
}

// performs funtion to configure the network access and policies
// before the container becomes active
func (d *Docker) AttachEndpoint(ctx *crtclient.ContainerEPContext) error {

	err := d.moveIfToContainer(ctx)
	if err != nil {
		return err
	}

	err = d.configureIfAddress(ctx)
	if err != nil {
		return err
	}

	// configure policies: acl/qos for the container on the host

	// cleanup intermediate things (overdoing it?)
	d.cleanupNetns(ctx)

	return err
}

// uninstall the policies and configuration during container attach
func (d *Docker) DetachEndpoint(ctx *crtclient.ContainerEPContext) error {
	var err error

	// log.Printf("Detached called for container %s with %s interface\n",
	//            ctx.CurrContName, ctx.InterfaceID)

	// no need to move the interface out of containre, etc.
	// usually deletion of ep takes care of that

	// TODO: unconfigure policies

	return err
}

func (d *Docker) GetContainerID(contName string) string {
	contInfo, err := d.Client.InspectContainer(contName)
	if err != nil {
		log.Printf("could not get contID for container %s, err '%s' \n",
			contName, err)
		return ""
	}

	// the hack below works only for running containers
	if !contInfo.State.Running {
		return ""
	}

	return contInfo.Id
}

func (d *Docker) GetContainerName(contID string) (string, error) {
	contInfo, err := d.Client.InspectContainer(contID)
	if err != nil {
		log.Printf("could not get contName for container %s, err '%s' \n",
			contID, err)
		return "", err
	}

	// the hack below works only for running containers
	if !contInfo.State.Running {
		return "", core.Errorf("container id %s not running", contID)
	}

	return contInfo.Name, nil
}
