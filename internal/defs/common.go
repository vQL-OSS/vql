/*
  The MIT License
  Copyright (c) 2020 FurtherSystem Co.,Ltd.

  Permission is hereby granted, free of charge, to any person obtaining a copy
  of this software and associated documentation files (the "Software"), to deal
  in the Software without restriction, including without limitation the rights
  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  copies of the Software, and to permit persons to whom the Software is
  furnished to do so, subject to the following conditions:

  The above copyright notice and this permission notice shall be included in
  all copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
  THE SOFTWARE.
*/

package defs

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net/url"
	"io"
	//"log"
	srand "crypto/rand"
	"encoding/json"
        "encoding/base64"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"golang.org/x/crypto/sha3"
)

// Version
var Version string

var ServiceCode = 0

var MagicKey = "KIWIKIWIKIWIKIWIKIWIKIWIKIWIKIWI"

// Initialize Rand , if you need fixed seed for some test case, noFixedSeed = false
func InitRand(noFixedSeed bool){
	if noFixedSeed {
		seed, _ := srand.Int(srand.Reader, big.NewInt(math.MaxInt64))
		rand.Seed(seed.Int64())
	} else {
		rand.Seed(1) // use fixed seed for test.
	}
}

// Create newguid - 16 bytes array
func NewGuid() ([16]byte, error) {
	uuid := [16]byte{}
	_, err := rand.Read(uuid[:])
	if err != nil {
		return uuid, err
	}
	return uuid, nil
}

// Get guid hexstring from 16 bytes array
func GuidFormatString(guid [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", guid[0:4], guid[4:6], guid[6:8], guid[8:10], guid[10:])
}

// Create new vendor code (base64 encodded)
func NewVendorCode() ([]byte, error) {
	hash := sha3.New256()
	guid, err := NewGuid()
	if err != nil {
		return nil, err
	}
	io.WriteString(hash, string(guid[:]))
	return hash.Sum(nil), nil
}

// Create new private code (base64 encodded)
func NewPrivateCode() ([]byte, error) {
	hash := sha3.New256()
	guid, err := NewGuid()
	if err != nil {
		return nil, err
	}
	io.WriteString(hash, string(guid[:]))
	return hash.Sum(nil), nil
}

// Create new session id (base64 encodded)
func NewSession() ([]byte, error) {
	hash := sha3.New256()
	guid, err := NewGuid()
	if err != nil {
		return nil, err
	}
	io.WriteString(hash, string(guid[:]))
	return hash.Sum(nil), nil
}

func ToBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// Create hash
func ToHash(b []byte) ([]byte, error) {
	hash := sha3.New256()
	io.WriteString(hash, string(b))
	return hash.Sum(nil), nil
}

func ToHmacSha256(msg, key string) string {
    mac := hmac.New(sha256.New, []byte(key))
    mac.Write([]byte(msg))
    return hex.EncodeToString(mac.Sum(nil))
}

// WebApi Payload Data Decode
func Decode(encoded []byte, v interface{}) error {
        urldecoded, err := url.QueryUnescape(string(encoded))
	if err != nil { return err}
        //log.Printf("base64 bytes: %x", []byte(urldecoded))
        decoded, err := base64.StdEncoding.DecodeString(urldecoded)
	if err != nil { return err}
        //log.Printf("encrypted bytes: %x", []byte(decoded))
        //var dest [256]byte
        //cipher, err := camellia.New(traditionalKey)
	//if err != nil { return err}
        //cipher.Decrypt(dest[:], decoded)
        //log.Printf("decoded bytes: %x", dest[:])
        //log.Printf("data: %s", string(dest[:]))
        return json.Unmarshal([]byte(decoded), v)
}
