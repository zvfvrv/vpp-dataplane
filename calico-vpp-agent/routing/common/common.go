// Copyright (C) 2020 Cisco Systems Inc.
// Copyright (C) 2016-2017 Nippon Telegraph and Telephone Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	bgpapi "github.com/osrg/gobgp/api"
	bgpserver "github.com/osrg/gobgp/pkg/server"
	calicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	calicocli "github.com/projectcalico/libcalico-go/lib/client"
	calicov3cli "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/vpp-dataplane/vpplink"
	"google.golang.org/protobuf/types/known/anypb"
	"net"
)

const (
	aggregatedPrefixSetBaseName = "aggregated"
	hostPrefixSetBaseName       = "host"
	policyBaseName              = "calico_aggr"
)

var (
	BgpFamilyUnicastIPv4 = bgpapi.Family{Afi: bgpapi.Family_AFI_IP, Safi: bgpapi.Family_SAFI_UNICAST}
	BgpFamilySRv6IPv4    = bgpapi.Family{Afi: bgpapi.Family_AFI_IP, Safi: bgpapi.Family_SAFI_SR_POLICY}
	BgpFamilyUnicastIPv6 = bgpapi.Family{Afi: bgpapi.Family_AFI_IP6, Safi: bgpapi.Family_SAFI_UNICAST}
	BgpFamilySRv6IPv6    = bgpapi.Family{Afi: bgpapi.Family_AFI_IP6, Safi: bgpapi.Family_SAFI_SR_POLICY}
)

// Data managed by the routing server and
// shared by the watchers
// This should be immutable for the lifetime
// of the agent
type RoutingData struct {
	Vpp                   *vpplink.VppLink
	BGPServer             *bgpserver.BgpServer
	HasV4                 bool
	HasV6                 bool
	Ipv4                  net.IP
	Ipv6                  net.IP
	Ipv4Net               *net.IPNet
	Ipv6Net               *net.IPNet
	Client                *calicocli.Client
	Clientv3              calicov3cli.Interface
	DefaultBGPConf        *calicov3.BGPConfigurationSpec
	ConnectivityEventChan chan ConnectivityEvent
}

type NodeState struct {
	Name      string
	Spec      calicov3.NodeSpec
	Status    calicov3.NodeStatus
	SweepFlag bool
}

func v46ify(s string, isv6 bool) string {
	if isv6 {
		return s + "-v6"
	} else {
		return s + "-v4"
	}
}

func GetPolicyName(isv6 bool) string {
	return v46ify(policyBaseName, isv6)
}

func GetAggPrefixSetName(isv6 bool) string {
	return v46ify(aggregatedPrefixSetBaseName, isv6)
}

func GetHostPrefixSetName(isv6 bool) string {
	return v46ify(hostPrefixSetBaseName, isv6)
}

func MakePath(prefix string, isWithdrawal bool, nodeIpv4 net.IP, nodeIpv6 net.IP) (*bgpapi.Path, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil, err
	}

	p := ipNet.IP
	masklen, _ := ipNet.Mask.Size()
	v4 := true
	if p.To4() == nil {
		v4 = false
	}

	nlri, err := ptypes.MarshalAny(&bgpapi.IPAddressPrefix{
		Prefix:    p.String(),
		PrefixLen: uint32(masklen),
	})
	if err != nil {
		return nil, err
	}
	var family *bgpapi.Family
	originAttr, err := ptypes.MarshalAny(&bgpapi.OriginAttribute{Origin: 0})
	if err != nil {
		return nil, err
	}
	attrs := []*any.Any{originAttr}

	if v4 {
		family = &BgpFamilyUnicastIPv4
		nhAttr, err := ptypes.MarshalAny(&bgpapi.NextHopAttribute{
			NextHop: nodeIpv4.String(),
		})
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, nhAttr)
	} else {
		family = &BgpFamilyUnicastIPv6
		nlriAttr, err := ptypes.MarshalAny(&bgpapi.MpReachNLRIAttribute{
			NextHops: []string{nodeIpv6.String()},
			Nlris:    []*any.Any{nlri},
			Family: &bgpapi.Family{
				Afi:  bgpapi.Family_AFI_IP6,
				Safi: bgpapi.Family_SAFI_UNICAST,
			},
		})
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, nlriAttr)
	}

	return &bgpapi.Path{
		Nlri:       nlri,
		IsWithdraw: isWithdrawal,
		Pattrs:     attrs,
		Age:        ptypes.TimestampNow(),
		Family:     family,
	}, nil
}

func MakePathSRv6(prefix string, isWithdrawal bool, nodeIpv4 net.IP, nodeIpv6 net.IP) (*bgpapi.Path, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil, err
	}

	p := ipNet.IP
	_, masklen_bits := ipNet.Mask.Size()
	v4 := true
	if p.To4() == nil {
		v4 = false
	}

	originAttr, err := ptypes.MarshalAny(&bgpapi.OriginAttribute{Origin: 0})
	if err != nil {
		return nil, err
	}
	attrs := []*any.Any{originAttr}
	//attrs := []*any.Any{originAttr, nh, rt, tun}

	var family *bgpapi.Family

	var nodeIP net.IP
	if v4 {
		nodeIP = nodeIpv4
	} else {
		nodeIP = nodeIpv6
	}
	nlrisr, err := ptypes.MarshalAny(&bgpapi.SRPolicyNLRI{
		Length:        uint32(masklen_bits),
		Distinguisher: 2,
		Color:         99,
		Endpoint:      p,
	})

	if err != nil {
		return nil, err
	}
	nhAttr, err := ptypes.MarshalAny(&bgpapi.NextHopAttribute{
		NextHop: nodeIP.String(),
	})
	if err != nil {
		return nil, err
	}
	attrs = append(attrs, nhAttr)

	// Tunnel Encapsulation Type 15 (SR Policy) sub tlvs
	s := make([]byte, 4)
	binary.BigEndian.PutUint32(s, 24321)
	sid, err := ptypes.MarshalAny(&bgpapi.SRBindingSID{
		SFlag: true,
		IFlag: false,
		Sid:   s,
	})
	if err != nil {
		return nil, err
	}
	bsid, err := ptypes.MarshalAny(&bgpapi.TunnelEncapSubTLVSRBindingSID{
		Bsid: sid,
	})
	if err != nil {
		return nil, err
	}
	segment, err := ptypes.MarshalAny(&bgpapi.SegmentTypeB{
		Flags: &bgpapi.SegmentFlags{
			SFlag: true,
		},
		Label: 10203,
	})
	if err != nil {
		return nil, err
	}
	seglist, err := ptypes.MarshalAny(&bgpapi.TunnelEncapSubTLVSRSegmentList{
		Weight: &bgpapi.SRWeight{
			Flags:  0,
			Weight: 12,
		},
		Segments: []*any.Any{segment},
	})
	if err != nil {
		return nil, err
	}
	pref, err := ptypes.MarshalAny(&bgpapi.TunnelEncapSubTLVSRPreference{
		Flags:      0,
		Preference: 11,
	})
	if err != nil {
		return nil, err
	}
	cpn, err := ptypes.MarshalAny(&bgpapi.TunnelEncapSubTLVSRCandidatePathName{
		CandidatePathName: "CandidatePathName",
	})
	if err != nil {
		return nil, err
	}
	pri, err := ptypes.MarshalAny(&bgpapi.TunnelEncapSubTLVSRPriority{
		Priority: 10,
	})
	if err != nil {
		return nil, err
	}
	// Tunnel Encapsulation attribute for SR Policy
	tun, err := ptypes.MarshalAny(&bgpapi.TunnelEncapAttribute{
		Tlvs: []*bgpapi.TunnelEncapTLV{
			{
				Type: 15,
				Tlvs: []*anypb.Any{bsid, seglist, pref, cpn, pri},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	attrs = append(attrs, tun)
	if v4 {
		family = &BgpFamilySRv6IPv4
	} else {
		family = &BgpFamilySRv6IPv6
	}

	return &bgpapi.Path{
		Nlri:      nlrisr,
		Pattrs:    attrs,
		Best:      true,
		SourceAsn: 64512,
		Family:    family,
	}, nil

	/*
		if v4 {
			family = &BgpFamilyUnicastIPv4
			nhAttr, err := ptypes.MarshalAny(&bgpapi.NextHopAttribute{
				NextHop: nodeIpv4.String(),
			})
			if err != nil {
				return nil, err
			}
			attrs = append(attrs, nhAttr)
		} else {
			family = &BgpFamilyUnicastIPv6
			nlriAttr, err := ptypes.MarshalAny(&bgpapi.MpReachNLRIAttribute{
				NextHops: []string{nodeIpv6.String()},
				Nlris:    []*any.Any{nlri},
				Family: &bgpapi.Family{
					Afi:  bgpapi.Family_AFI_IP6,
					Safi: bgpapi.Family_SAFI_UNICAST,
				},
			})
			if err != nil {
				return nil, err
			}
			attrs = append(attrs, nlriAttr)
		}*/
	/*
		&bgpapi.Path{
			Nlri:      nlrisr,
			IsWithdraw: isWithdrawal,
			Pattrs:    attrs,
			Age:        ptypes.TimestampNow(),
			Family:    &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_SR_POLICY},
			Best:      true,
			SourceAsn: 65000,
		}
	*/

}

type ChangeType int

const (
	ChangeNone    ChangeType = iota
	ChangeSame    ChangeType = iota
	ChangeAdded   ChangeType = iota
	ChangeDeleted ChangeType = iota
	ChangeUpdated ChangeType = iota
)

func GetStringChangeType(old, new string) ChangeType {
	if old == new && new == "" {
		return ChangeNone
	} else if old == new {
		return ChangeSame
	} else if old == "" {
		return ChangeAdded
	} else if new == "" {
		return ChangeDeleted
	} else {
		return ChangeUpdated
	}
}

type ConnectivityEventType string

const (
	NodeStateChanged   ConnectivityEventType = "NodeStateChanged"
	FelixConfChanged   ConnectivityEventType = "FelixConfChanged"
	IpamConfChanged    ConnectivityEventType = "IpamConfChanged"
	VppRestart         ConnectivityEventType = "VppRestart"
	RescanState        ConnectivityEventType = "RescanState"
	ConnectivtyAdded   ConnectivityEventType = "ConnectivtyAdded"
	ConnectivtyDeleted ConnectivityEventType = "ConnectivtyDeleted"
)

type ConnectivityEvent struct {
	Type ConnectivityEventType

	Old interface{}
	New interface{}
}

type NodeConnectivity struct {
	Dst              net.IPNet
	NextHop          net.IP
	ResolvedProvider string
}

func (cn *NodeConnectivity) String() string {
	return fmt.Sprintf("%s-%s", cn.Dst.String(), cn.NextHop.String())
}
