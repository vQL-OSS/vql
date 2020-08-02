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
	"crypto/hmac"
	srand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io"
	"math"
	"math/big"
	"math/rand"
	"net/url"
	"runtime"
	"github.com/labstack/echo/v4"
)

const stacktraceDepth = 2
type ResponseCode int16
var VendorSeed string
var SessionSeed string

const (
	// Common XX0XX
	ResponseOk                        ResponseCode = 0  // ok, success
	ResponseNgDefault                              = 1  // ng, failed provisional error. should not use this code.
	ResponseOkContinue                             = -1 // ok, polling continue.
	ResponseNgSecSquashed                          = 2  // ng, cannot open details for security.
	ResponseNgServerTimeout                        = 10 // ng, server timeout.
	ResponseNgClientTimeout                        = 11 // ng, client timeout.
	ResponseNgEncodeInvalid                        = 12 // ng, encode failed.
	ResponseNgHashGenerateFailed                   = 13 // ng, hash generate failed.
	ResponseNgShardConnectFailed                   = 14 // ng, db shard connect failed.
	ResponseNgTransactBeginFailed                  = 15 // ng, db transaction failed.
	ResponseNgPreparedStatementFailed              = 16 // ng, db prepared statement create failed.
	ResponseNgQueryExecuteFailed                   = 17 // ng, db query execute failed.
	ResponseNgRollbackFailed                       = 18 // ng, db rollback failed.
	ResponseNgCommitFailed                         = 19 // ng, db commit failed.
	// VendorRegist XX1XX
	ResponseNgVendorNameBlank      = 100 // ng, vendor name is blank.
	ResponseNgVendorNameMaxover    = 101 // ng, vendor name is capacity over.
	ResponseNgVendorNameInvalid    = 102 // ng, vendor name is invalid. bad caractor.
	ResponseNgVendorCaptionMaxover = 103 // ng, vendor caption is capacity over.
	ResponseNgVendorCaptionInvalid = 104 // ng, vendor caption is invalid. bad caractor.
	ResponseNgNonceInvalid         = 105 // ng, nonce invalid.
	ResponseNgTicksInvalid         = 106 // ng, ticks invalid.
	ResponseNgSeedInvalid          = 107 // ng, seed verify failed.
	// VendorRegist RecoverCode XX1XX
	ResponseOkVendorAccountBadPrivkeyExistsSeed = -100 // ok, bad private key but, exists seed.
	ResponseOkVendorAccountBadPrivkeyExistsSso  = -101 // ok, bad private key but, exists sso.
	ResponseOkVendorAccountNoPrivkeyExistsSeed  = -102 // ok, no private key but, exists seed.
	ResponseOkVendorAccountNoPrivkeyExistsSso   = -103 // ok, no private key but, exists sso.
	ResponseOkVendorAccountRecoveredDataInvalid = -110 // ok, vendor account invalid data recovered.
	ResponseOkVendorAccountRecoveredFromSeed    = -111 // ok, vendor account recovered from seed.
	ResponseOkVendorAccountRecoveredFromSso     = -112 // ok, vendor account recovered from sso.
	// VendorView XX2XX
	ResponseNgVendorConnotMoveup     = 200 // ng, this is top, cannot more moveup
	ResponseNgVendorAlreadyShelved   = 201 // ng, already sheleved.
	ResponseNgVendorAlreadyUnshelved = 202 // ng, already unshelved.
	ResponseNgVendorAlreadyCanceled  = 203 // ng, already canceled by vendor.
	// VendorDequeueAuth XX3XX
	ResponseNgVendorCannotAuthDequeue = 300 // ng, user dequeue auth not executed, dequeue auth failed.
	// VendorNotify XX4XX
	// VendorAuthOption XX5XX
	ResponseNgVendorAuthLacked = 500 // ng, vendor auth info lacked.
	ResponseNgVendorAuthFailed = 501 // ng, vendor auth failed.
	// UserQueing XX6XX
	ResponseNgUserMaxover   = 600 // ng, user cannot queing, user max over.
	ResponseNgUserOutoftime = 601 // ng, user cannot queing, out of time.
	// UserView XX7XX
	ResponseNgUserAlreadyMailOn   = 700 // ng, user already mail on.
	ResponseNgUserAlreadyMailOff  = 701 // ng, user already mail off.
	ResponseNgUserAlreadyPushOn   = 702 // ng, user already push on.
	ResponseNgUserAlreadyPushOff  = 703 // ng, user already push off.
	ResponseNgUserCannotPending   = 704 // ng, this is end. cannot more pending.
	ResponseNgUserAlreadyCanceled = 705 // ng, already canceled by user.
	// UserDequeueAuth XX8XX
	ResponseNgUserCannotAuthDequeue = 800 // ng, vendor dequeue auth not executed on time, dequeue auth failed.
	// UserAuthOption XX9XX
	ResponseNgUserAuthLacked = 900 // ng, user auth info lacked.
	ResponseNgUserAuthFailed = 901 // ng, user auth failed.
	ResponseNgUserAuthNotFound = 902 // ng, user auth not found.
)

var responseCodeText = map[ResponseCode]string{
	ResponseOk:                                  "ResponseOk",
	ResponseNgDefault:                           "ResponseNgDefault",
	ResponseOkContinue:                          "ResponseOkContinue",
	ResponseNgSecSquashed:                       "ResponseNgSecSquashed",
	ResponseNgServerTimeout:                     "ResponseNgServerTimeout",
	ResponseNgClientTimeout:                     "ResponseNgClientTimeout",
	ResponseNgEncodeInvalid:                     "ResponseNgEncodeInvalid",
	ResponseNgHashGenerateFailed:                "ResponseNgHashGenerateFailed",
	ResponseNgShardConnectFailed:                "ResponseNgShardConnectFailed",
	ResponseNgTransactBeginFailed:               "ResponseNgTransactBeginFailed",
	ResponseNgPreparedStatementFailed:           "ResponseNgPreparedStatementFailed",
	ResponseNgQueryExecuteFailed:                "ResponseNgQueryExecuteFailed",
	ResponseNgRollbackFailed:                    "ResponseNgRollbackFailed",
	ResponseNgCommitFailed:                      "ResponseNgCommitFailed",
	ResponseNgVendorNameBlank:                   "ResponseNgVendorNameBlank",
	ResponseNgVendorNameMaxover:                 "ResponseNgVendorNameMaxover",
	ResponseNgVendorNameInvalid:                 "ResponseNgVendorNameInvalid",
	ResponseNgVendorCaptionMaxover:              "ResponseNgVendorCaptionMaxover",
	ResponseNgVendorCaptionInvalid:              "ResponseNgVendorCaptionInvalid",
	ResponseNgNonceInvalid:                      "ResponseNgNonceInvalid",
	ResponseNgTicksInvalid:                      "ResponseNgTicksInvalid",
	ResponseNgSeedInvalid:                       "ResponseNgSeedInvalid",
	ResponseOkVendorAccountBadPrivkeyExistsSeed: "ResponseOkVendorAccountBadPrivkeyExistsSeed",
	ResponseOkVendorAccountBadPrivkeyExistsSso:  "ResponseOkVendorAccountBadPrivkeyExistsSso",
	ResponseOkVendorAccountNoPrivkeyExistsSeed:  "ResponseOkVendorAccountNoPrivkeyExistsSeed",
	ResponseOkVendorAccountNoPrivkeyExistsSso:   "ResponseOkVendorAccountNoPrivkeyExistsSso",
	ResponseOkVendorAccountRecoveredDataInvalid: "ResponseOkVendorAccountRecoveredDataInvalid",
	ResponseOkVendorAccountRecoveredFromSeed:    "ResponseOkVendorAccountRecoveredFromSeed",
	ResponseOkVendorAccountRecoveredFromSso:     "ResponseOkVendorAccountRecoveredFromSso",
	ResponseNgVendorConnotMoveup:                "ResponseNgVendorConnotMoveup",
	ResponseNgVendorAlreadyShelved:              "ResponseNgVendorAlreadyShelved",
	ResponseNgVendorAlreadyUnshelved:            "ResponseNgVendorAlreadyUnshelved",
	ResponseNgVendorAlreadyCanceled:             "ResponseNgVendorAlreadyCanceled",
	ResponseNgVendorCannotAuthDequeue:           "ResponseNgVendorCannotAuthDequeue",
	ResponseNgVendorAuthLacked:                  "ResponseNgVendorAuthLacked",
	ResponseNgVendorAuthFailed:                  "ResponseNgVendorAuthFailed",
	ResponseNgUserMaxover:                       "ResponseNgUserMaxover",
	ResponseNgUserOutoftime:                     "ResponseNgUserOutoftime",
	ResponseNgUserAlreadyMailOn:                 "ResponseNgUserAlreadyMailOn",
	ResponseNgUserAlreadyMailOff:                "ResponseNgUserAlreadyMailOff",
	ResponseNgUserAlreadyPushOn:                 "ResponseNgUserAlreadyPushOn",
	ResponseNgUserAlreadyPushOff:                "ResponseNgUserAlreadyPushOff",
	ResponseNgUserCannotPending:                 "ResponseNgUserCannotPending",
	ResponseNgUserAlreadyCanceled:               "ResponseNgUserAlreadyCanceled",
	ResponseNgUserCannotAuthDequeue:             "ResponseNgUserCannotAuthDequeue",
	ResponseNgUserAuthLacked:                    "ResponseNgUserAuthLacked",
	ResponseNgUserAuthFailed:                    "ResponseNgUserAuthFailed",
	ResponseNgUserAuthNotFound:                  "ResponseNgUserAuthNotFound",
}

type AuthContext struct {
        echo.Context
        Uid            uint64
}

func ResponseCodeText(c ResponseCode) string {
	return responseCodeText[c]
}

// Initialize Rand , if you need fixed seed for some test case, noFixedSeed = false
func InitRand(noFixedSeed bool) {
	if noFixedSeed {
		seed, _ := srand.Int(srand.Reader, big.NewInt(math.MaxInt64))
		rand.Seed(seed.Int64())
	} else {
		rand.Seed(1) // use fixed seed for test.
	}
}

func InitSeed() {
	VendorSeedBytes, _ := NewGuid()
	SessionSeedBytes, _ := NewGuid()
	VendorSeed = string(VendorSeedBytes[:])
	SessionSeed = string(SessionSeedBytes[:])
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
	// TODO migrate to VendorSeed
	io.WriteString(hash, string(guid[:]))
	return hash.Sum(nil), nil
}

// Create new queue code (base64 encodded)
func NewQueueCode() ([]byte, error) {
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
func NewSession(keyword string) ([]byte, error) {
	hash := sha3.New256()
	io.WriteString(hash, SessionSeed + keyword)
	return hash.Sum(nil), nil
}

// Create new session private code (base64 encodded)
func NewSessionPrivate() ([]byte, error) {
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
func Decode(encoded []byte, v interface{}, t int64) error {
	urldecoded, err := url.QueryUnescape(string(encoded))
	if err != nil {
		return err
	}
	//log.Printf("base64 bytes: %x", []byte(urldecoded))
	decoded, err := base64.StdEncoding.DecodeString(urldecoded)
	if err != nil {
		return err
	}
	//log.Printf("encrypted bytes: %x", []byte(decoded))
	//var dest [256]byte
	//cipher, err := camellia.New(traditionalKey)
	//if err != nil { return err}
	//cipher.Decrypt(dest[:], decoded)
	//log.Printf("decoded bytes: %x", dest[:])
	//log.Printf("data: %s", string(dest[:]))
	return json.Unmarshal([]byte(decoded), v)
}

// WebApi Payload Data Encode
func Encode(v interface{}, t int64) string {
	jsonData, err := json.Marshal(v)
	if err != nil {
		//log.Printf("encode error : %s", err.Error())
		return ""
	}
	//log.Printf("jsonData bytes: %s", string(jsonData))
	data := base64.StdEncoding.EncodeToString([]byte(jsonData))
	//log.Printf("data bytes: %s", data)
	//escData := url.QueryEscape(string(jsonData))
	//log.Printf("escData bytes: %s", escData)
	return data
}

// request body base struct
type MessageBodyBase struct {
	Ticks int64 `json:Ticks`
}

func (m *MessageBodyBase) GetTicks() int64 {
	return m.Ticks
}

type MessageHandle interface {
	GetTicks() int64
}

// request body base struct
type RequestBodyBase struct {
	MessageBodyBase
}

type RequestHandle interface {
	GetTicks() int64
}

// response body base struct
type ResponseBodyBase struct {
	ResponseCode ResponseCode `json:ResponseCode`
	MessageBodyBase
}

func (m *ResponseBodyBase) SetResponseCode(c ResponseCode) {
	m.ResponseCode = c
}

type ResponseHandle interface {
	GetTicks() int64
	SetResponseCode(c ResponseCode)
}

func ErrorDispose(cx echo.Context, r ResponseHandle, c ResponseCode, securitySquash bool, e error) string {
	cx.Logger().Debugf("response code %s : %s", ResponseCodeText(c), e.Error())
	if !ProdMode {
		printStacktrace(cx)
	}
	if securitySquash {
		r.SetResponseCode(ResponseNgSecSquashed)
	} else {
		r.SetResponseCode(c)
	}
	return Encode(r, r.GetTicks())
}

func printStacktrace(cx echo.Context) {
	stackMax := 20
	for stack := 0; stack < stackMax; stack++ {
		if stack < stacktraceDepth {
			continue
		}
		point, file, line, ok := runtime.Caller(stack)
		if !ok {
			break
		}
		funcName := runtime.FuncForPC(point).Name()
		cx.Logger().Debugf("[STACKTRACE] file=%s, line=%d, func=%v\n", file, line, funcName)
	}
}
