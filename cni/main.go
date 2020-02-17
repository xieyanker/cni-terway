package main

import (
	"context"
	"encoding/json"
	"net"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"k8s.io/klog"

	"github.com/generals-space/cni-terway/netconf"
)

var versionAll = version.PluginSupports("0.3.1")

// AddRouteInPod 在Pod空间里添加到ServiceIP的路由, 需要设置宿主机为该Pod的网关.
func AddRouteInPod(bridgeName, netnsPath, serviceIPCIDR string) (err error) {
	linkBridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		klog.Errorf("faliled to get bridge link: %s", err)
		return
	}
	bridgeAddrs, err := netlink.AddrList(linkBridge, netlink.FAMILY_V4)
	if err != nil {
		klog.Errorf("faliled to get bridge link: %s", err)
		return
	}
	klog.V(3).Infof("bridge addrs: %+v, len: %d", bridgeAddrs, len(bridgeAddrs))
	var gw net.IP
	if len(bridgeAddrs) > 0 {
		gw = bridgeAddrs[0].IP
	}

	svcRoute := &netlink.Route{
		Dst: &net.IPNet{
			IP: net.IPv4(10, 96, 0, 0), Mask: net.CIDRMask(12, 32),
		},
		Gw: gw,
	}
	if serviceIPCIDR != "" {
		// ParseCIDR 解析 192.168.0.0/12 网络字符串
		_, cidr, err := net.ParseCIDR(serviceIPCIDR)
		if err != nil {
			klog.Errorf("failed to parse service ip cidr %s: %v", serviceIPCIDR, err)
			return err
		}
		svcRoute.Dst = cidr
	}

	netns, err := ns.GetNS(netnsPath)
	if err != nil {
		klog.Errorf("failed to open netns %q: %v", netnsPath, err)
		return err
	}
	defer netns.Close()

	return netns.Do(func(containerNS ns.NetNS) (err error) {
		link, err := netlink.LinkByName("eth0")
		if err != nil {
			klog.Errorf("faliled to get eth0 link: %s", err)
			return err
		}
		svcRoute.LinkIndex = link.Attrs().Index
		err = netlink.RouteAdd(svcRoute)
		if err != nil {
			klog.Errorf("faliled to add route: %s", err)
			return err
		}
		return nil
	})
}

// cmdAdd: 在调用此函数时, 以由kubelet创建好pause容器, 正是需要为其部署网络的时候.
// 而对应的业务容器此时还未创建.
func cmdAdd(args *skel.CmdArgs) (err error) {
	klog.V(3).Infof("cmdAdd args: %+v", args)
	netConf := &netconf.NetConf{}
	err = json.Unmarshal(args.StdinData, netConf)
	if err != nil {
		return
	}
	delegateBytes, err := json.Marshal(netConf.Delegate)
	if err != nil {
		return
	}

	///////////////////////////////////////////////////
	result, err := invoke.DelegateAdd(context.TODO(), netConf.Delegate["type"].(string), delegateBytes, nil)
	if err != nil {
		klog.Errorf("faliled to run bridge plugin: %s", err)
		return
	}
	klog.Infof("run bridge plugin success: %s", result.Print())

	err = AddRouteInPod(netConf.Delegate["bridge"].(string), args.Netns, netConf.ServiceIPCIDR)
	if err != nil {
		klog.Errorf("faliled to add route to the pod %s: %s", args.Args, err)
		return
	}

	return
}

func cmdDel(args *skel.CmdArgs) error {
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	// TODO: implement
	return nil
}

func main() {
	klog.Info("start cni-terway plugin...")
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, versionAll, "cni-terway")
}
