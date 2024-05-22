// Package gdt manages Global Descriptor Table.
//
// http://www.osdever.net/bkerndev/Docs/gdt.htm
package gdt

import (
	"math"
	"unsafe"
)

var (
	Table = [3]Entry{}
	GP    Ptr
)

// Comptime check that Table's size does not overflow uint16.
var _ = [math.MaxUint16]struct{}{}[math.MaxUint16-unsafe.Sizeof(Table)]

type Entry struct {
	limitLow    uint16
	baseLow     uint16
	baseMiddle  uint8
	access      uint8
	granularity uint8
	baseHigh    uint8
}

type Ptr struct {
	limit uint16
	base  uint32
}

func Init() {
	GP.limit = uint16(unsafe.Sizeof(Table)) - 1
	GP.base = uint32(uintptr(unsafe.Pointer(&Table)))

	// NULL descriptor
	Table[0] = newEntry(0, 0, 0, 0)
	/* The second entry is our Code Segment. The base address
	 *  is 0, the limit is 4GBytes, it uses 4KByte granularity,
	 *  uses 32-bit opcodes, and is a Code Segment descriptor.
	 *  Please check the table above in the tutorial in order
	 *  to see exactly what each value means */
	Table[1] = newEntry(0, ^uint32(0), 0x9A, 0xCF)
	/* The third entry is our Data Segment. It's EXACTLY the
	 *  same as our code segment, but the descriptor type in
	 *  this entry's access byte says it's a Data Segment */
	Table[2] = newEntry(0, ^uint32(0), 0x92, 0xCF)

	flush(&GP)
}

func newEntry(base uint32, limit uint32, access uint8, gran uint8) Entry {
	return Entry{
		baseLow:     uint16(base & 0xFFFF),
		baseMiddle:  uint8(base >> 16),
		baseHigh:    uint8(base >> 24),
		limitLow:    uint16(limit & 0xFFFF),
		granularity: uint8(limit>>16) | (gran & 0xF0),
		access:      access,
	}
}

func flush(*Ptr)
