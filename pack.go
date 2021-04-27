package main

//#include <stdio.h>
//#include <stdint.h>
//struct Packed_Struct {
//  uint16_t A;
//  uint16_t B;
//  uint32_t C;
//  uint16_t D;
//} __attribute__((packed)) packs;
//
//struct UnPacked_Struct {
//  uint16_t A;
//  uint16_t B;
//  uint32_t C;
//  uint16_t D;
//} upacks;
//
//
//void print_C_struct_size(){
//  struct Packed_Struct Packed_Struct;
//  struct UnPacked_Struct UnPacked_Struct;
//  printf("Sizeof Packed_Struct: %lu\n", sizeof(Packed_Struct) );
//  printf("Sizeof UnPacked_Struct: %lu\n", sizeof(UnPacked_Struct) );
//  return;
//}
//
import "C"

import(
    "fmt"
    "unsafe"
	"bytes"
	"encoding/binary"
)

type GoStruct struct{
    A   uint16 //12
    B   uint16 //34
    C   uint32 //5678
    D   uint16 //9A
}

func main(){
	meh := C.print_C_struct_size()
	cpacks := C.packs
    fmt.Printf("Sizeof PackedStruct from go : %d\n", unsafe.Sizeof(cpacks))  //doesnt work	
    var GoStruct GoStruct
    fmt.Printf("Sizeof GoStruct : %d\n", unsafe.Sizeof(GoStruct) )
	
    fmt.Printf("meh type: %T\n", meh)
	fmt.Printf("struct type: %T\n", cpacks)

	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.LittleEndian, GoStruct)
	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Sizeof GoStruct: %d, Sizeof buf: %d, Len of buf: %d\n", unsafe.Sizeof(GoStruct), unsafe.Sizeof(buf), buf.Len() )
	
}
