// Diode Network Client
// Copyright 2019 IoT Blockchain Technology Corporation LLC (IBTC)
// Licensed under the Diode License, Version 1.0
package util

import (
	"bytes"
	"fmt"
	"testing"
)

type IsHexTest struct {
	Src []byte
	Res bool
}

type IsHexNumberTest struct {
	Src []byte
	Res bool
}

type IsAddressTest struct {
	Src []byte
	Res bool
}

type IsSubdomainTest struct {
	Src string
	Res bool
}

type DecodeStringTest struct {
	Src string
	Res []byte
}

type DecodeBytesIntTest struct {
	Src []byte
	Res int
}

type DecodeBytesUintTest struct {
	Src []byte
	Res uint64
}

var (
	isHexTest = []IsHexTest{
		IsHexTest{
			Src: []byte{1},
			Res: false,
		},
		IsHexTest{
			Src: []byte("0x1234"),
			Res: true,
		},
		IsHexTest{
			Src: []byte("0X1234"),
			Res: false,
		},
		IsHexTest{
			Src: []byte("0xzxvn"),
			Res: false,
		},
	}
	isHexNumberTest = []IsHexNumberTest{
		IsHexNumberTest{
			Src: []byte{1},
			Res: false,
		},
		IsHexNumberTest{
			Src: []byte("0x1234"),
			Res: false,
		},
		IsHexNumberTest{
			Src: []byte("0X1234"),
			Res: true,
		},
		IsHexNumberTest{
			Src: []byte("0Xljhg"),
			Res: false,
		},
	}
	decodeStringTest = []DecodeStringTest{
		DecodeStringTest{
			Src: "0x01",
			Res: []byte{1},
		},
		DecodeStringTest{
			Src: "0x1234",
			Res: []byte{18, 52},
		},
	}
	isAddressTest = []IsAddressTest{
		IsAddressTest{
			Src: []byte{1},
			Res: false,
		},
		IsAddressTest{
			Src: []byte("0x937c492a77ae90de971986d003ffbc5f8bb2232C"),
			Res: true,
		},
		IsAddressTest{
			Src: []byte("0x937c492a77ae90de971986d003ffbc5f8bb2232c"),
			Res: true,
		},
		IsAddressTest{
			Src: []byte("0X937c492a77ae90de971986d003ffbc5f8bb2232c"),
			Res: false,
		},
		IsAddressTest{
			Src: []byte("937c492a77ae90de971986d003ffbc5f8bb2232c"),
			Res: false,
		},
	}
	decodeBytesIntTest = []DecodeBytesIntTest{
		DecodeBytesIntTest{
			Src: []byte{1},
			Res: 1,
		},
		DecodeBytesIntTest{
			Src: []byte{10},
			Res: 10,
		},
		DecodeBytesIntTest{
			Src: []byte{1, 0},
			Res: 256,
		},
		DecodeBytesIntTest{
			Src: []byte{1, 1, 0},
			Res: 65792,
		},
	}
	decodeBytesUintTest = []DecodeBytesUintTest{
		DecodeBytesUintTest{
			Src: []byte{1},
			Res: 1,
		},
		DecodeBytesUintTest{
			Src: []byte{10},
			Res: 10,
		},
		DecodeBytesUintTest{
			Src: []byte{1, 0},
			Res: 256,
		},
		DecodeBytesUintTest{
			Src: []byte{1, 1, 0},
			Res: 65792,
		},
	}
	isSubdomainTest = []IsSubdomainTest{
		IsSubdomainTest{
			Src: "0x937c492a77ae90de971986d003ffbc5f8bb2232C",
			Res: true,
		},
		IsSubdomainTest{
			Src: "937c492a77ae90de971986d003ffbc5f8bb2232C",
			Res: false,
		},
		IsSubdomainTest{
			Src: "Helloworld",
			Res: true,
		},
		IsSubdomainTest{
			Src: "Hello-world",
			Res: true,
		},
		IsSubdomainTest{
			Src: "Hell/oworld",
			Res: false,
		},
		IsSubdomainTest{
			Src: "Hell&oworld",
			Res: false,
		},
		IsSubdomainTest{
			Src: "Hell%oworld",
			Res: false,
		},
		IsSubdomainTest{
			Src: "Hell&oworld",
			Res: false,
		},
		IsSubdomainTest{
			Src: "Hell_oworld",
			Res: false,
		},
		IsSubdomainTest{
			Src: "Hell=oworld",
			Res: false,
		},
	}
)

func TestIsHex(t *testing.T) {
	for _, v := range isHexTest {
		if v.Res != IsHex(v.Src) {
			t.Errorf("Wrong result when call IsHex")
		}
	}
}

func TestIsHexNumber(t *testing.T) {
	for _, v := range isHexNumberTest {
		if v.Res != IsHexNumber(v.Src) {
			t.Errorf("Wrong result when call IsHexNumber")
		}
	}
}

func TestIsAddress(t *testing.T) {
	for _, v := range isAddressTest {
		if v.Res != IsAddress(v.Src) {
			t.Errorf("Wrong result when call IsAddress")
		}
	}
}

func TestIsSubdomain(t *testing.T) {
	for _, v := range isSubdomainTest {
		if v.Res != IsSubdomain(v.Src) {
			t.Errorf("Wrong result when call IsSubdomain")
		}
	}
}

func TestDecodeString(t *testing.T) {
	for _, v := range decodeStringTest {
		res, _ := DecodeString(v.Src)
		if !bytes.Equal(v.Res, res) {
			t.Errorf("Wrong result when call DecodeString")
		}
	}
}

func TestDecodeBytesToInt(t *testing.T) {
	for _, v := range decodeBytesIntTest {
		res := DecodeBytesToInt(v.Src)
		if v.Res != res {
			t.Errorf("Wrong result when call DecodeBytesToInt")
		}
	}
}

func TestDecodeBytesToUint(t *testing.T) {
	for _, v := range decodeBytesUintTest {
		res := DecodeBytesToUint(v.Src)
		if v.Res != res {
			t.Errorf("Wrong result when call DecodeBytesToUint")
		}
	}
}

func TestDecodeIntToBytes(t *testing.T) {
	for _, v := range decodeBytesIntTest {
		res := DecodeIntToBytes(v.Res)
		if !bytes.Equal(v.Src, res) {
			t.Errorf("Wrong result when call DecodeIntToBytes")
		}
	}
}

func TestDecodeInt64ToBytes(t *testing.T) {
	for _, v := range decodeBytesIntTest {
		res := DecodeInt64ToBytes(int64(v.Res))
		if !bytes.Equal(v.Src, res) {
			t.Errorf("Wrong result when call DecodeInt64ToBytes")
		}
	}
}

func TestEncodeToString(t *testing.T) {
	for _, v := range decodeStringTest {
		res := EncodeToString(v.Res)
		if v.Src != res {
			t.Errorf("Wrong result when call EncodeToString")
		}
	}
}

func TestEncodeForce(t *testing.T) {
	for _, v := range decodeStringTest {
		res := fmt.Sprintf("0x%s", string(EncodeForce(v.Res)))
		if v.Src != res {
			t.Errorf("Wrong result when call EncodeToString")
		}
	}
}
