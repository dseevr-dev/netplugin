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

package netutils

import (
	"testing"
)

type testSubnetInfo struct {
	subnetIP  string
	subnetLen uint
	hostId    uint
	hostIP    string
}

var testSubnets = []testSubnetInfo{
	{subnetIP: "11.2.1.0", subnetLen: 24, hostId: 5, hostIP: "11.2.1.5"},
	{subnetIP: "10.123.16.0", subnetLen: 22, hostId: 513, hostIP: "10.123.18.1"},
	{subnetIP: "172.12.0.0", subnetLen: 16, hostId: 261, hostIP: "172.12.1.5"},
}

func TestGetSubnetIP(t *testing.T) {
	for _, te := range testSubnets {
		hostIP, err := GetSubnetIP(te.subnetIP, te.subnetLen, 32, te.hostId)
		if err != nil {
			t.Fatalf("error getting host ip from subnet %s/%d for hostid %d - err '%s'",
				te.subnetIP, te.subnetLen, te.hostId, err)
		}
		if hostIP != te.hostIP {
			t.Fatalf("obtained ip %s doesn't match expected ip %s for subnet %s/%d\n",
				hostIP, te.hostIP, te.subnetIP, te.subnetLen)
		}
	}
}

var testInvalidSubnets = []testSubnetInfo{
	{subnetIP: "11.2.1.0", subnetLen: 32, hostId: 5, hostIP: "11.2.1.5"},
	{subnetIP: "10.123.16.0", subnetLen: 22, hostId: 1025, hostIP: "10.123.18.1"},
	{subnetIP: "172.12.0.0", subnetLen: 4, hostId: 261, hostIP: "172.12.1.5"},
}

func TestInvalidGetSubnetIP(t *testing.T) {
	for _, te := range testInvalidSubnets {
		_, err := GetSubnetIP(te.subnetIP, te.subnetLen, 32, te.hostId)
		if err == nil {
			t.Fatalf("Expecting error on invalid config subnet %s/%d for hostid %d",
				te.subnetIP, te.subnetLen, te.hostId)
		}
	}
}

func TestGetIPNumber(t *testing.T) {
	for _, te := range testSubnets {
		hostId, err := GetIPNumber(te.subnetIP, te.subnetLen, 32, te.hostIP)
		if err != nil {
			t.Fatalf("error getting host ip from subnet %s/%d for hostid %d ",
				te.subnetIP, te.subnetLen, te.hostId)
		}
		if hostId != te.hostId {
			t.Fatalf("obtained ip %d doesn't match with expected ip %d \n",
				hostId, te.hostId)
		}
	}
}

func TestValidRange(t *testing.T) {
	rangeStr := "5-100, 101-200"
	_, err := ParseTagRanges(rangeStr, "vlan")
	if err != nil {
		t.Fatalf("error '%s' parsing valid vlan range '%s'\n", err, rangeStr)
	}
}

func TestInvalidVlanRange(t *testing.T) {
	rangeStr := "5--100, 101-200"
	_, err := ParseTagRanges(rangeStr, "vlan")
	if err == nil {
		t.Fatalf("successfully parsed invalid vlan range '%s'\n", rangeStr)
	}
}

func TestInvalidVlanValue(t *testing.T) {
	rangeStr := "5-100, 101-5000"
	_, err := ParseTagRanges(rangeStr, "vlan")
	if err == nil {
		t.Fatalf("successfully parsed invalid vlan id '%s'\n", rangeStr)
	}
}

func TestInvalidMinMaxVlan(t *testing.T) {
	rangeStr := "5-100, 200-101"
	_, err := ParseTagRanges(rangeStr, "vlan")
	if err == nil {
		t.Fatalf("successfully parsed invalid min-max vlan values '%s'\n", rangeStr)
	}
}

func TestInvalidRangeExtraSeperators(t *testing.T) {
	rangeStr := "5-100,,101-200"
	_, err := ParseTagRanges(rangeStr, "vlan")
	if err == nil {
		t.Fatalf("successfully parsed vlan range with extra seperators '%s'\n", rangeStr)
	}
}

func TestValidVxlanRange(t *testing.T) {
	rangeStr := "10000-16000"
	_, err := ParseTagRanges(rangeStr, "vxlan")
	if err != nil {
		t.Fatalf("error '%s' parsing valid vxlan range '%s'\n", err, rangeStr)
	}
}

func TestInvalidVxlanMultipleRanges(t *testing.T) {
	rangeStr := "101-400, 10000-15000"
	_, err := ParseTagRanges(rangeStr, "vxlan")
	if err == nil {
		t.Fatalf("successfully parsed invalid vxlan value '%s'\n", rangeStr)
	}
}

func TestInvalidVxlanValue(t *testing.T) {
	rangeStr := "101-75535"
	_, err := ParseTagRanges(rangeStr, "vxlan")
	if err == nil {
		t.Fatalf("successfully parsed invalid vxlan value '%s'\n", rangeStr)
	}
}

func TestInvalidMinMaxVxlan(t *testing.T) {
	rangeStr := "8000-7999"
	_, err := ParseTagRanges(rangeStr, "vxlan")
	if err == nil {
		t.Fatalf("successfully parsed invalid min-max vxlan values '%s'\n", rangeStr)
	}
}

type testSubnetAllocInfo struct {
	subnetIP       string
	subnetLen      uint
	subnetAllocLen uint
	hostId         uint
	hostIP         string
}

var testSubnetAllocs = []testSubnetAllocInfo{
	{subnetIP: "11.1.0.0", subnetLen: 16, subnetAllocLen: 24,
		hostId: 5, hostIP: "11.1.5.0"},
	{subnetIP: "10.0.0.0", subnetLen: 8, subnetAllocLen: 24,
		hostId: 5, hostIP: "10.0.5.0"},
}

func TestGetSubnetAlloc(t *testing.T) {
	for _, te := range testSubnetAllocs {
		hostIP, err := GetSubnetIP(te.subnetIP, te.subnetLen,
			te.subnetAllocLen, te.hostId)
		if err != nil {
			t.Fatalf("error getting subnet ip from subnet-range %s/%d alloc-size %d "+
				"for id %d - err '%s'",
				te.subnetIP, te.subnetLen, te.subnetAllocLen, te.hostId, err)
		}
		if hostIP != te.hostIP {
			t.Fatalf("obtained ip %s doesn't match expected ip %s for subnet %s/%d "+
				"for AllocLen %d \n",
				hostIP, te.hostIP, te.subnetIP, te.subnetLen, te.subnetAllocLen)
		}
	}
}

func TestGetSubnetNumber(t *testing.T) {
	for _, te := range testSubnetAllocs {
		hostId, err := GetIPNumber(te.subnetIP, te.subnetLen,
			te.subnetAllocLen, te.hostIP)
		if err != nil {
			t.Fatalf("error getting subnet ip from subnet %s/%d for hostid %d "+
				"for subnet alloc size %d \n",
				te.subnetIP, te.subnetLen, te.hostId, te.subnetAllocLen)
		}
		if hostId != te.hostId {
			t.Fatalf("obtained subnet ip %d doesn't match with expected ip %d "+
				"for subnet %s/%d alloc size %d \n",
				hostId, te.hostId, te.subnetIP, te.subnetLen, te.subnetAllocLen)
		}
	}
}

func TestGetLocalIP(t *testing.T) {
	ipAddr, err := GetLocalIP()
	if ipAddr == "" {
		t.Fatalf("error obtaining local IP of the host '%s' \n", err)
	}
}
