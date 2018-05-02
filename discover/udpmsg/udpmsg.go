/*
 *  Copyright (C) 2017 gyee authors
 *
 *  This file is part of the gyee library.
 *
 *  the gyee library is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  the gyee library is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with the gyee library.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package udpmsg

import (
	"net"
	yclog	"ycp2p/logger"
	ycfg	"ycp2p/config"
	pb		"ycp2p/discover/udpmsg/pb"
)

//
// UDP messages for discovering protocol tasks
//
const (
	UdpMsgTypePing		= iota
	UdpMsgTypePong
	UdpMsgTypeFindNode
	UdpMsgTypeNeighbors
	UdpMsgTypeUnknown
)

type UdpMsgType int

type (
	// Endpoint
	Endpoint struct {
		IP			net.IP
		UDP			uint16
		TCP			uint16
	}

	// Node: endpoint with node identity
	Node struct {
		IP			net.IP
		UDP			uint16
		TCP			uint16
		NodeId		ycfg.NodeID
	}

	// Ping
	Ping struct {
		From		Node
		To			Node
		Expiration	uint64
		Id			uint64
		Extra		[]byte
	}

	// Pong: response to Ping
	Pong struct {
		From		Node
		To			Node
		Id			uint64
		Expiration	uint64
		Extra		[]byte
	}

	// FindNode: request the endpoint of the target
	FindNode struct {
		From		Node
		To			Node
		Target		Node
		Id			uint64
		Expiration	uint64
		Extra		[]byte
	}

	// Neighbors: response to FindNode
	Neighbors struct {
		From		Node
		To			Node
		Id			uint64
		Nodes		[]*Node
		Expiration	uint64
		Extra		[]byte
	}
)

//
// UDP message: tow parts, the first is the raw bytes ... the seconde is
// protobuf message. for decoding, protobuf message will be extract from
// the raw one; for encoding, bytes will be wriiten into raw buffer.
//
// Notice: since we would only one UDP reader for descovering, we can put
// an UdpMsg instance here.
//
type UdpMsg struct {
	Buf		[]byte
	Len		int
	From	*net.UDPAddr
	Msg		pb.UdpMessage
	Eno		UdpMsgErrno
}

var udpMsg = UdpMsg {
	Buf:	nil,
	Len:	0,
	From:	nil,
	Msg:	nil,
	Eno:	UdpMsgEnoUnknown,
}

var PtrUdpMsg = &udpMsg

const (
	UdpMsgEnoNone 		= iota
	UdpMsgEnoParameter
	UdpMsgEnoEncodeFailed
	UdpMsgEnoDecodeFailed
	UdpMsgEnoMessage
	UdpMsgEnoUnknown
)

type UdpMsgErrno int

//
// Set raw message
//
func (pum *UdpMsg) SetRawMessage(buf []byte, len int, from *net.UDPAddr) UdpMsgErrno {
	if buf == nil || len == 0 || from == nil {
		yclog.LogCallerFileLine("SetRawMessage: invalid parameter(s)")
		return UdpMsgEnoParameter
	}
	pum.Eno = UdpMsgEnoNone
	pum.Buf = buf
	pum.Len = len
	pum.From = from
	return UdpMsgEnoNone
}

//
// Decoding
//
func (pum *UdpMsg) Decode() UdpMsgErrno {
	if err := (&pum.Msg).Unmarshal(pum.Buf); err != nil {
		yclog.LogCallerFileLine("Decode: Unmarshal failed, err: %s", err.Error())
		return UdpMsgEnoDecodeFailed
	}
	return UdpMsgEnoNone
}

//
// Get decoded message
//
func (pum *UdpMsg) GetPbMessage() *pb.UdpMessage {
	return &pum.Msg
}

//
// Get decoded message
//
func (pum *UdpMsg) GetDecodedMsg() interface{} {

	// get type
	mt := pum.GetDecodedMsgType()
	if mt == UdpMsgTypeUnknown {
		yclog.LogCallerFileLine("GetDecodedMsg: GetDecodedMsgType failed, mt: %d", mt)
		return nil
	}

	// map type to function and the get
	var funcMap = map[UdpMsgType]interface{} {
		UdpMsgTypePing: pum.GetPing,
		UdpMsgTypePong: pum.GetPong,
		UdpMsgTypeFindNode: pum.GetFindNode,
		UdpMsgTypeNeighbors: pum.GetNeighbors,
	}

	var f interface{}
	var ok bool
	if f, ok = funcMap[mt]; !ok {
		yclog.LogCallerFileLine("GetDecodedMsg: invalid message type: %d", mt)
		return nil
	}

	return f.(func()interface{})()
}

//
// Get deocded message type
//
func (pum *UdpMsg) GetDecodedMsgType() UdpMsgType {
	var pbMap = map[pb.UdpMessage_MessageType]UdpMsgType {
		pb.UdpMessage_PING:			UdpMsgTypePing,
		pb.UdpMessage_PONG:			UdpMsgTypePong,
		pb.UdpMessage_FINDNODE:		UdpMsgTypePong,
		pb.UdpMessage_NEIGHBORS:	UdpMsgTypePing,
	}

	var key pb.UdpMessage_MessageType
	var val UdpMsgType
	var ok bool
	key = pum.Msg.GetMsgType()
	if val, ok = pbMap[key]; !ok {
		yclog.LogCallerFileLine("GetDecodedMsgType: invalid message type")
		return UdpMsgTypeUnknown
	}

	return val
}

//
// Get decoded Ping
//
func (pum *UdpMsg) GetPing() *Ping {
	pbPing := pum.Msg.Ping
	ping := new(Ping)

	var inf interface{}
	ping.From.IP = pbPing.From.IP
	ping.From.TCP = uint16(*pbPing.From.TCP)
	ping.From.UDP = uint16(*pbPing.From.UDP)
	inf = pbPing.From.NodeId
	ping.From.NodeId = *inf.(*ycfg.NodeID)

	ping.To.IP = pbPing.To.IP
	ping.To.TCP = uint16(*pbPing.To.TCP)
	ping.To.UDP = uint16(*pbPing.To.UDP)
	inf = pbPing.To.NodeId
	ping.To.NodeId = *inf.(*ycfg.NodeID)

	ping.Expiration = *pbPing.Expiration
	ping.Extra = pbPing.Extra

	return ping
}

//
// Get decoded Pong
//
func (pum *UdpMsg) GetPong() *Pong {
	pbPong := pum.Msg.Pong
	pong := new(Pong)

	var inf interface{}
	pong.From.IP = pbPong.From.IP
	pong.From.TCP = uint16(*pbPong.From.TCP)
	pong.From.UDP = uint16(*pbPong.From.UDP)
	inf = pbPong.From.NodeId
	pong.From.NodeId = *inf.(*ycfg.NodeID)

	pong.To.IP = pbPong.To.IP
	pong.To.TCP = uint16(*pbPong.To.TCP)
	pong.To.UDP = uint16(*pbPong.To.UDP)
	inf = pbPong.To.NodeId
	pong.To.NodeId = *inf.(*ycfg.NodeID)

	pong.Expiration = *pbPong.Expiration
	pong.Extra = pbPong.Extra

	return pong
}

//
// Get decoded FindNode
//
func (pum *UdpMsg) GetFindNode() *FindNode {
	pbFN := pum.Msg.FindNode
	fn := new(FindNode)

	var inf interface{}
	fn.From.IP = pbFN.From.IP
	fn.From.TCP = uint16(*pbFN.From.TCP)
	fn.From.UDP = uint16(*pbFN.From.UDP)
	inf = pbFN.From.NodeId
	fn.From.NodeId = *inf.(*ycfg.NodeID)

	fn.To.IP = pbFN.To.IP
	fn.To.TCP = uint16(*pbFN.To.TCP)
	fn.To.UDP = uint16(*pbFN.To.UDP)
	inf = pbFN.To.NodeId
	fn.To.NodeId = *inf.(*ycfg.NodeID)

	fn.Target.IP = pbFN.Target.IP
	fn.Target.TCP = uint16(*pbFN.Target.TCP)
	fn.Target.UDP = uint16(*pbFN.Target.UDP)
	inf = pbFN.Target.NodeId
	fn.Target.NodeId = *inf.(*ycfg.NodeID)

	fn.Expiration = *pbFN.Expiration
	fn.Extra = pbFN.Extra

	return fn
}

//
// Get decoded Neighbors
//
func (pum *UdpMsg) GetNeighbors() *Neighbors {
	pbNgb := pum.Msg.Neighbors
	ngb := new(Neighbors)

	var inf interface{}
	ngb.From.IP = pbNgb.From.IP
	ngb.From.TCP = uint16(*pbNgb.From.TCP)
	ngb.From.UDP = uint16(*pbNgb.From.UDP)
	inf = pbNgb.From.NodeId
	ngb.From.NodeId = *inf.(*ycfg.NodeID)

	ngb.To.IP = pbNgb.To.IP
	ngb.To.TCP = uint16(*pbNgb.To.TCP)
	ngb.To.UDP = uint16(*pbNgb.To.UDP)
	inf = pbNgb.To.NodeId
	ngb.To.NodeId = *inf.(*ycfg.NodeID)

	ngb.Expiration = *pbNgb.Expiration
	ngb.Extra = pbNgb.Extra

	for idx, n := range pbNgb.Nodes {
		ngb.Nodes[idx].IP = n.IP
		ngb.Nodes[idx].TCP = uint16(*n.TCP)
		ngb.Nodes[idx].UDP = uint16(*n.UDP)
		inf = n.NodeId
		ngb.Nodes[idx].NodeId = *inf.(*ycfg.NodeID)
	}

	return ngb
}

//
// Check decoded message with endpoint where the message from
//
func (pum *UdpMsg) CheckUdpMsgFromPeer(from *net.UDPAddr) bool {

	// we just check the ip address simply now, more might be needed
	funcBytesEqu := func(bys1[]byte, bys2[]byte) bool {
		if len(bys1) != len(bys2) {
			return false
		}
		for idx, b := range bys1 {
			if b != bys2[idx] {
				return false
			}
		}
		return true
	}

	if *pum.Msg.MsgType == pb.UdpMessage_PING {
		return funcBytesEqu(pum.Msg.Ping.From.IP, from.IP)
	} else if *pum.Msg.MsgType == pb.UdpMessage_PONG  {
		return funcBytesEqu(pum.Msg.Pong.From.IP, from.IP)
	} else if *pum.Msg.MsgType == pb.UdpMessage_FINDNODE {
		return funcBytesEqu(pum.Msg.FindNode.From.IP, from.IP)
	} else if *pum.Msg.MsgType == pb.UdpMessage_NEIGHBORS {
		return funcBytesEqu(pum.Msg.Neighbors.From.IP, from.IP)
	}

	return false
}

//
// Encode directly from protobuf message
//
func (pum *UdpMsg) EncodePbMsg() UdpMsgErrno {
	var err error
	if pum.Buf, err = (&pum.Msg).Marshal(); err != nil {
		yclog.LogCallerFileLine("Encode: Marshal failed, err: %s", err.Error())
		pum.Eno = UdpMsgEnoEncodeFailed
		return pum.Eno
	}
	pum.Eno = UdpMsgEnoNone
	return pum.Eno
}

//
// Encode from UDP messages
//
func (pum *UdpMsg) Encode(t int, msg interface{}) UdpMsgErrno {

	var eno UdpMsgErrno

	switch t {
	case UdpMsgTypePing:
		eno = pum.EncodePing(msg.(*Ping))
		break
	case UdpMsgTypePong:
		eno = pum.EncodePong(msg.(*Pong))
		break
	case UdpMsgTypeFindNode:
		eno = pum.EncodeFindNode(msg.(*FindNode))
		break
	case UdpMsgTypeNeighbors:
		eno = pum.EncodeNeighbors(msg.(*Neighbors))
		break
	default:
		eno = UdpMsgEnoParameter
	}

	if eno != UdpMsgEnoNone {
		yclog.LogCallerFileLine("Encode: failed, type: %d", t)
	}

	pum.Eno = eno
	return eno
}

//
// Encode Ping
//
func (pum *UdpMsg) EncodePing(ping *Ping) UdpMsgErrno {
	var pbm = &pum.Msg
	var pbPing *pb.UdpMessage_Ping

	pbm.MsgType = new(pb.UdpMessage_MessageType)
	*pbm.MsgType = pb.UdpMessage_PING
	pbPing = new(pb.UdpMessage_Ping)

	var inf interface{}
	pbPing.From.IP = ping.From.IP
	*pbPing.From.TCP = uint32(ping.From.TCP)
	*pbPing.From.UDP = uint32(ping.From.UDP)
	inf = ping.From.NodeId
	pbPing.From.NodeId = inf.([]byte)

	pbPing.To.IP = ping.To.IP
	*pbPing.To.TCP = uint32(ping.To.TCP)
	*pbPing.To.UDP = uint32(ping.To.UDP)
	inf = ping.To.NodeId
	pbPing.To.NodeId = inf.([]byte)

	*pbPing.Expiration = ping.Expiration
	pbPing.Extra = ping.Extra

	var err error
	var buf []byte
	if buf, err = pbPing.Marshal(); err != nil {
		yclog.LogCallerFileLine("EncodePing: fialed, err: %s", err.Error())
		return UdpMsgEnoEncodeFailed
	}
	pum.Buf = buf
	pum.Len = len(buf)
	pum.Msg.Ping = pbPing

	return UdpMsgEnoNone
}

//
// Encode Pong
//
func (pum *UdpMsg) EncodePong(pong *Pong) UdpMsgErrno {
	var pbm = &pum.Msg
	var pbPong *pb.UdpMessage_Pong

	pbm.MsgType = new(pb.UdpMessage_MessageType)
	*pbm.MsgType = pb.UdpMessage_PONG
	pbPong = new(pb.UdpMessage_Pong)

	var inf interface{}
	pbPong.From.IP = pong.From.IP
	*pbPong.From.TCP = uint32(pong.From.TCP)
	*pbPong.From.UDP = uint32(pong.From.UDP)
	inf = pong.From.NodeId
	pbPong.From.NodeId = inf.([]byte)

	pbPong.To.IP = pong.To.IP
	*pbPong.To.TCP = uint32(pong.To.TCP)
	*pbPong.To.UDP = uint32(pong.To.UDP)
	inf = pong.To.NodeId
	pbPong.To.NodeId = inf.([]byte)

	*pbPong.Expiration = pong.Expiration
	pbPong.Extra = pong.Extra

	var err error
	var buf []byte
	if buf, err = pbPong.Marshal(); err != nil {
		yclog.LogCallerFileLine("EncodePong: fialed, err: %s", err.Error())
		return UdpMsgEnoEncodeFailed
	}
	pum.Buf = buf
	pum.Len = len(buf)
	pum.Msg.Pong = pbPong

	return UdpMsgEnoNone
}

//
// Encode FindNode
//
func (pum *UdpMsg) EncodeFindNode(fn *FindNode) UdpMsgErrno {
	var pbm = &pum.Msg
	var pbFN *pb.UdpMessage_FindNode

	pbm.MsgType = new(pb.UdpMessage_MessageType)
	*pbm.MsgType = pb.UdpMessage_PONG
	pbFN = new(pb.UdpMessage_FindNode)

	var inf interface{}
	pbFN.From.IP = fn.From.IP
	*pbFN.From.TCP = uint32(fn.From.TCP)
	*pbFN.From.UDP = uint32(fn.From.UDP)
	inf = fn.From.NodeId
	pbFN.From.NodeId = inf.([]byte)

	pbFN.To.IP = fn.To.IP
	*pbFN.To.TCP = uint32(fn.To.TCP)
	*pbFN.To.UDP = uint32(fn.To.UDP)
	inf = fn.To.NodeId
	pbFN.To.NodeId = inf.([]byte)

	pbFN.Target.IP = fn.Target.IP
	*pbFN.Target.TCP = uint32(fn.Target.TCP)
	*pbFN.Target.UDP = uint32(fn.Target.UDP)
	inf = fn.Target.NodeId
	pbFN.Target.NodeId = inf.([]byte)

	*pbFN.Expiration = fn.Expiration
	pbFN.Extra = fn.Extra

	var err error
	var buf []byte
	if buf, err = pbFN.Marshal(); err != nil {
		yclog.LogCallerFileLine("EncodeFindNode: fialed, err: %s", err.Error())
		return UdpMsgEnoEncodeFailed
	}
	pum.Buf = buf
	pum.Len = len(buf)
	pum.Msg.FindNode = pbFN

	return UdpMsgEnoNone
}

//
// Encode Neighbors
//
func (pum *UdpMsg) EncodeNeighbors(ngb *Neighbors) UdpMsgErrno {
	var pbm = &pum.Msg
	var pbNgb *pb.UdpMessage_Neighbors

	pbm.MsgType = new(pb.UdpMessage_MessageType)
	*pbm.MsgType = pb.UdpMessage_NEIGHBORS
	pbNgb = new(pb.UdpMessage_Neighbors)

	var inf interface{}
	pbNgb.From.IP = ngb.From.IP
	*pbNgb.From.TCP = uint32(ngb.From.TCP)
	*pbNgb.From.UDP = uint32(ngb.From.UDP)
	inf = ngb.From.NodeId
	pbNgb.From.NodeId = inf.([]byte)

	pbNgb.To.IP = ngb.To.IP
	*pbNgb.To.TCP = uint32(ngb.To.TCP)
	*pbNgb.To.UDP = uint32(ngb.To.UDP)
	inf = ngb.To.NodeId
	pbNgb.To.NodeId = inf.([]byte)

	*pbNgb.Expiration = ngb.Expiration
	pbNgb.Extra = ngb.Extra

	for idx, n := range ngb.Nodes {
		pbNgb.Nodes[idx].IP = n.IP
		*pbNgb.Nodes[idx].TCP = uint32(n.TCP)
		*pbNgb.Nodes[idx].UDP = uint32(n.UDP)
		inf = n.NodeId
		pbNgb.Nodes[idx].NodeId = inf.([]byte)
	}

	var err error
	var buf []byte
	if buf, err = pbNgb.Marshal(); err != nil {
		yclog.LogCallerFileLine("EncodeNeighbors: fialed, err: %s", err.Error())
		return UdpMsgEnoEncodeFailed
	}
	pum.Buf = buf
	pum.Len = len(buf)
	pum.Msg.Neighbors = pbNgb

	return UdpMsgEnoNone
}

//
// Get buffer and length of bytes for message encoded
//
func (pum *UdpMsg) GetRawMessage() (buf []byte, len int) {
	if pum.Eno != UdpMsgEnoNone {
		return nil, 0
	}
	return pum.Buf, pum.Len
}

//
// Compare two nodes
//
const (
	CmpNodeEqu		= iota
	CmpNodeNotEquId
	CmpNodeNotEquIp
	CmpNodeNotEquUdpPort
	CmpNodeNotEquTcpPort
)

func (n1 *Node) CompareWith(n2 *Node) int {
	if n1.NodeId != n2.NodeId {
		return CmpNodeNotEquId
	} else if n1.IP.Equal(n2.IP) != true {
		return CmpNodeNotEquIp
	} else if n1.UDP != n2.UDP {
		return CmpNodeNotEquUdpPort
	} else if n1.TCP != n2.TCP {
		return CmpNodeNotEquTcpPort
	}
	return CmpNodeEqu
}
