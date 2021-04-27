package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
	"math/rand"
)

/*
    0                   1                   2                   3
    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |V=2|P|X|  CC   |M|     PT      |       sequence number         |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                           timestamp                           |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |           synchronization source (SSRC) identifier            |
   +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
   |            contributing source (CSRC) identifiers             |
   |                             ....                              |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

func createHeader(seq uint16, ts, ssrc uint32) []byte {

	//bigendian composition

	header := make([]byte, 4*3) // 12byte
	header[0] |= 0x80 // version 2              v_p_x_cc
	header[1] |= 0x00 // G.711 ulaw             m_pt
	binary.BigEndian.PutUint16(header[2:], seq) //?? big or little
	binary.BigEndian.PutUint32(header[4:], ts) //?? big or little
	binary.BigEndian.PutUint32(header[8:], ssrc) //?? big or little
	//csrc if needed.
	return header
}


func main() {

	if false {	
		var i uint32
		buf := bytes.NewReader([]byte{0x12, 0x34, 0x56, 0x78, 0x12})
		if err := binary.Read(buf, binary.BigEndian, &i); err != nil {
			fmt.Println("1 binary.Read failed:", err)
		}
		fmt.Printf("0x%x\n", i)

		buf.Seek(0,0)

		if err := binary.Read(buf, binary.LittleEndian, &i); err != nil {
			fmt.Println("2 binary.Read failed:", err)
		}
		fmt.Printf("0x%x\n", i)

		const(
			A uint = 10   // 1010
			B uint = 12   // 1100
		)

		var bits uint

		// AND演算
		bits = A & B   // 1000
		fmt.Printf("%4b\n", bits)
		// OR演算
		bits = A | B   // 1110
		fmt.Printf("%4b\n", bits)	
		// XOR演算
		bits = A ^ B   // 0110
		fmt.Printf("%04b\n", bits)	
		// AND NOT演算
		bits = A &^ B  // 0010  1010 からB=1100をマスクとして前半の11部分をクリア、後半の00部分はそのまま残す
		fmt.Printf("%04b %v\n", bits, bits)	
		// 左シフト演算
		bits = 1 << uint64(3) // 1000 : 2の3乗かかる
		fmt.Printf("%04b\n", bits)
		// 右シフト演算
		bits = 8 >> uint64(3) // 0001 : 2の(-3)乗かかる
		fmt.Printf("%04b\n", bits)
	}

	if false {

		header := make([]byte, 12) //[]byte{0x00, 0x00, 0x00, 0x00}
		binary.LittleEndian.PutUint32(header[0:], 0x2)
		fmt.Printf("%s", hex.Dump(header))			

		header = make([]byte, 12)
		binary.LittleEndian.PutUint16(header[1:], 0x8FF7)
		fmt.Printf("%s", hex.Dump(header))		

		header = make([]byte, 12)	
		binary.BigEndian.PutUint32(header[0:], 0x2)
		fmt.Printf("%s", hex.Dump(header))	

		header = make([]byte, 12)
		binary.BigEndian.PutUint16(header[1:], 0x8FF7)
		fmt.Printf("%s", hex.Dump(header))

	}

	var t uint32 = uint32(time.Now().Unix())
	rand.Seed(time.Now().UnixNano())
	var ssrc = rand.Uint32()

	fmt.Printf("%x, %x\n", t, ssrc)
	b := createHeader(0x1, t, ssrc)
	fmt.Printf("%s", hex.Dump(b))
	
}
