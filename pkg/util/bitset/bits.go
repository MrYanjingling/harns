package bitset

import (
	"fmt"
	"k8s.io/klog/v2"
	"math/bits"
	"strings"
)

const (
	addressBitsPerWord uint   = 6
	bitsPerWord        uint   = 1 << addressBitsPerWord
	bitIndexMask       uint   = bitsPerWord - 1
	allBits            uint64 = 0xffffffffffffffff
)

type BitSet struct {
	len uint
	set []uint64
}

func New(n uint) *BitSet {
	if n <= 0 {
		klog.V(2).InfoS("Invalid BitSet length", "length", n)
		return nil
	}
	return &BitSet{
		len: n,
		set: make([]uint64, ((n-1)>>addressBitsPerWord)+1),
	}
}

func (b *BitSet) Len() uint {
	return b.len
}

func (b *BitSet) SetValue(i uint, value bool) {
	if i >= b.len {
		return
	}
	if value {
		b.Set(i)
	} else {
		b.Clear(i)
	}
}

func (b *BitSet) Get(i uint) bool {
	if i >= b.len {
		return false
	}
	return b.set[i>>addressBitsPerWord]&(1<<(i&bitIndexMask)) != 0
}

func (b *BitSet) Clean() {
	for i := range b.set {
		b.set[i] = 0
	}
}

func (b *BitSet) Set(i uint) {
	if i >= b.len {
		return
	}
	b.set[i>>addressBitsPerWord] |= 1 << (i & bitIndexMask)
}

func (b *BitSet) Clear(i uint) {
	if i >= b.len {
		return
	}
	b.set[i>>addressBitsPerWord] &^= 1 << (i & bitIndexMask)
}

// NextSet get the index of the first bit that is set to true
// that occurs on or after the specified starting index
func (b *BitSet) NextSet(i uint) (uint, bool) {
	x := int(i >> addressBitsPerWord)
	if x >= len(b.set) {
		return 0, false
	}
	w := b.set[x]
	w = w >> (i & bitIndexMask)
	if w != 0 {
		return i + uint(bits.TrailingZeros64(w)), true
	}
	x++
	for x < len(b.set) {
		if b.set[x] != 0 {
			return uint(x)*bitsPerWord + uint(bits.TrailingZeros64(b.set[x])), true
		}
		x++
	}
	return 0, false
}

// NextClear get the index of the first bit that is set to false
// that occurs on or after the specified starting index
func (b *BitSet) NextClear(i uint) (uint, bool) {
	x := int(i >> addressBitsPerWord)
	if x >= len(b.set) {
		return 0, false
	}
	w := b.set[x]
	w = w >> (i & bitIndexMask)
	wA := allBits >> (i & bitIndexMask)
	index := i + uint(bits.TrailingZeros64(^w))
	if w != wA && index < b.len {
		return index, true
	}
	x++
	for x < len(b.set) {
		index = uint(x)*bitsPerWord + uint(bits.TrailingZeros64(^b.set[x]))
		if b.set[x] != allBits && index < b.len {
			return index, true
		}
		x++
	}
	return 0, false
}

func (b *BitSet) String() string {
	sb := strings.Builder{}
	quotient, remainder := b.len/bitsPerWord, b.len%bitsPerWord

	fmts := "%0" + fmt.Sprintf("%d", remainder) + "b"
	sb.WriteString(fmt.Sprintf(fmts, b.set[quotient]))

	for i := int(quotient) - 1; i >= 0; i-- {
		sb.WriteString(fmt.Sprintf("%064b", b.set[i]))
	}
	return sb.String()
}
