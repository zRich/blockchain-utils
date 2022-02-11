package main

import (
	"fmt"
	"math/big"
	"time"
	"unsafe"
)

var (
	bigOne = big.NewInt(1)
	// 最大难度：00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff，2^224，0x1d00ffff
	mainPowLimit      = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)
	powTargetTimespan = time.Hour * 24 * 14 // 两周
)

type BlockHeader struct {
	Height    uint32
	Bits      uint32
	Timestamp time.Time `json:",omitempty"`
}
type Block struct {
	Header BlockHeader
}

func CompactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

func BigToCompact(n *big.Int) uint32 {
	// No need to do any work if it's zero.
	if n.Sign() == 0 {
		return 0
	}

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes.  So, shift the number right or left
	// accordingly.  This is equivalent to:
	// mantissa = mantissa / 256^(exponent-3)
	var mantissa uint32
	exponent := uint(len(n.Bytes()))
	if exponent <= 3 {
		mantissa = uint32(n.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		// Use a copy to avoid modifying the caller's original number.
		tn := new(big.Int).Set(n)
		mantissa = uint32(tn.Rsh(tn, 8*(exponent-3)).Bits()[0])
	}

	// When the mantissa already has the sign bit set, the number is too
	// large to fit into the available 23-bits, so divide the number by 256
	// and increment the exponent accordingly.
	if mantissa&0x00800000 != 0 {
		mantissa >>= 8
		exponent++
	}

	// Pack the exponent, sign bit, and mantissa into an unsigned 32-bit
	// int and return it.
	compact := uint32(exponent<<24) | mantissa
	if n.Sign() < 0 {
		compact |= 0x00800000
	}
	return compact
}

func CalculateNextWorkTarget(prev2016block, lastBlock Block) *big.Int {
	// 如果新区块(+1)不是2016的整数倍，则不需要更新，仍然是最后一个区块的 bits
	if (lastBlock.Header.Height+1)%2016 != 0 {
		return CompactToBig(lastBlock.Header.Bits)
	}
	// 计算 2016个区块出块时间
	actualTimespan := lastBlock.Header.Timestamp.Sub(prev2016block.Header.Timestamp)
	if actualTimespan < powTargetTimespan/4 {
		actualTimespan = powTargetTimespan / 4
	} else if actualTimespan > powTargetTimespan*4 {
		// 如果超过8周，则按8周计算
		actualTimespan = powTargetTimespan * 4
	}
	lastTarget := CompactToBig(lastBlock.Header.Bits)
	// 计算公式： target = lastTarget * actualTime / expectTime
	newTarget := new(big.Int).Mul(lastTarget, big.NewInt(int64(actualTimespan.Seconds())))
	newTarget.Div(newTarget, big.NewInt(int64(powTargetTimespan.Seconds())))
	//超过最多难度，则重置
	if newTarget.Cmp(mainPowLimit) > 0 {
		newTarget.Set(mainPowLimit)
	}
	return newTarget
}

func main() {
	firstTime, _ := time.Parse("2006-01-02 15:04:05", "2017-11-25 03:53:16")
	lastTime, _ := time.Parse("2006-01-02 15:04:05", "2017-12-07 00:22:42")
	prevB := Block{Header: BlockHeader{Height: 497951, Bits: 0x1800d0f6, Timestamp: lastTime}}
	prev2016B := Block{Header: BlockHeader{Height: 495936, Bits: 0x1800d0f6, Timestamp: firstTime}}
	result := CalculateNextWorkTarget(prev2016B, prevB)

	bits := BigToCompact(result)
	fmt.Printf("Result in hex %064x\n", bits)
	if bits != 0x1800b0ed {
		fmt.Printf("expect 0x1800b0ed,unexpected %x\n", bits)
	}

	n := uint32(0x170b8c8b)

	fmt.Printf("Target Hash in Hex %064x\n", CompactToBig(n))
}
