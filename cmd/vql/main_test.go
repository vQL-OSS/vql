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

// Consumer side web api package
package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"io/ioutil"
	"strings"
	"testing"
	"vql/internal/defs"
	"vql/internal/db"
	"vql/internal/routes"
	"vql/internal/routes/priv"
	"vql/internal/routes/queue"
	"vql/internal/routes/vendor"
)

// Enqueue test no require admit
func TestEnqueueNoRequireAdmit(t *testing.T) {
	defs.ServicePrefix = "gotest"
	assert.NoError(t, db.Setup())
	assert.NoError(t, db.Conns.Init())
	assert.NoError(t, db.OpConns.Init())
	e := echo.New()
	route.Init(e)
	e.Logger.SetLevel(log.DEBUG)

	reqBody := queue.ReqBodyCreate{}
	reqBody.IdentifierType = 0
	reqBody.Identifier = "57ea5c1f17211a2c384a05030a88fcace73d9d92bd1c714da5c68ede09847304"
	reqBody.Seed = "9c463571a92614f5ed8ff55c249e7b8c458860e030284d2b5bcc7a529ac58741"
	reqBody.Ticks = 1592619000
	base64Encoded := defs.Encode(reqBody, reqBody.Ticks)
	urlEncoded := url.QueryEscape(base64Encoded)
	req := httptest.NewRequest(http.MethodPost, "/new", strings.NewReader(urlEncoded))
	rec := httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	c := e.NewContext(req, rec)

	assert.NoError(t, queue.Create(c))
        resCreate := queue.ResBodyCreate{}
        bodyBytes, _ := ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resCreate, resCreate.Ticks);

        reqLogon := queue.ReqBodyLogon{}
        reqLogon.PrivateCode = resCreate.PrivateCode
        reqLogon.Ticks = 1592619000
        base64Encoded = defs.Encode(reqLogon, reqLogon.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	c = e.NewContext(req, rec)
	assert.NoError(t, queue.Logon(c))

	reqUpgrade := vendor.ReqBodyUpgrade{}
	reqUpgrade.Name = "vendor sample"
	reqUpgrade.Caption = "caption sample"
	reqUpgrade.Ticks = 1592619000
	base64Encoded = defs.Encode(reqUpgrade, reqUpgrade.Ticks)
	urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/vendor/upgrade", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx := &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.Upgrade(authCtx))
        resUpgrade := vendor.ResBodyUpgrade{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resUpgrade, resUpgrade.Ticks);

	// new queue no require admit
        reqInitQueue := vendor.ReqBodyInitQueue{}
        reqInitQueue.RequireAdmit = false
        reqInitQueue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqInitQueue, reqInitQueue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/vendor/queue/new", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.InitQueue(authCtx))
        resInitQueue := vendor.ResBodyInitQueue{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resInitQueue, resInitQueue.Ticks);

        reqEnqueue := queue.ReqBodyEnqueue{}
        reqEnqueue.VendorCode = resUpgrade.VendorCode
        reqEnqueue.QueueCode = resInitQueue.QueueCode
        reqEnqueue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqEnqueue, reqEnqueue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/queue", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, queue.Enqueue(authCtx))
        resEnqueue := queue.ResBodyEnqueue{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resEnqueue, resEnqueue.Ticks);

	// show queue
        vendorCodeUrlUnsafed := resUpgrade.VendorCode
        queueCodeUrlUnsafed := resInitQueue.QueueCode
        r := strings.NewReplacer("=", "-", "/", "_", "+", ".")
        vendorCodeUrlSafed := r.Replace(vendorCodeUrlUnsafed)
        queueCodeUrlSafed := r.Replace(queueCodeUrlUnsafed)
	req = httptest.NewRequest(http.MethodGet, "/on/queue/"+vendorCodeUrlSafed+"/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/queue/:vendor_code/:queue_code")
	c.SetParamNames("vendor_code", "queue_code")
	c.SetParamValues(vendorCodeUrlSafed, queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, queue.ShowQueue(authCtx))

	// show queue vendor
	req = httptest.NewRequest(http.MethodGet, "/on/vendor/queue/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/queue/:queue_code")
	c.SetParamNames("queue_code")
	c.SetParamValues(queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.ShowQueue(authCtx))

	// manage queue vendor 
	req = httptest.NewRequest(http.MethodGet, "/on/vendor/manage/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/manage/:queue_code")
	c.SetParamNames("queue_code")
	c.SetParamValues(queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.Manage(authCtx))

	// dequeue by user
        reqDequeue := queue.ReqBodyDequeue{}
        reqDequeue.KeyCodePrefix = resEnqueue.KeyCodePrefix
        reqDequeue.KeyCodeSuffix = resEnqueue.KeyCodeSuffix
        reqDequeue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqDequeue, reqDequeue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/dequeue", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, queue.Dequeue(authCtx))

	assert.NoError(t, priv.DropVendor(authCtx))
	assert.NoError(t, db.Teardown())
}

// Enqueue test require admit polite dequeue
func TestEnqueueRequireAdmitPolite(t *testing.T) {
	defs.ServicePrefix = "gotest"
	assert.NoError(t, db.Setup())
	assert.NoError(t, db.Conns.Init())
	assert.NoError(t, db.OpConns.Init())
	e := echo.New()
	route.Init(e)
	e.Logger.SetLevel(log.DEBUG)

	reqBody := queue.ReqBodyCreate{}
	reqBody.IdentifierType = 0
	reqBody.Identifier = "57ea5c1f17211a2c384a05030a88fcace73d9d92bd1c714da5c68ede09847304"
	reqBody.Seed = "9c463571a92614f5ed8ff55c249e7b8c458860e030284d2b5bcc7a529ac58741"
	reqBody.Ticks = 1592619000
	base64Encoded := defs.Encode(reqBody, reqBody.Ticks)
	urlEncoded := url.QueryEscape(base64Encoded)
	req := httptest.NewRequest(http.MethodPost, "/new", strings.NewReader(urlEncoded))
	rec := httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	c := e.NewContext(req, rec)

	assert.NoError(t, queue.Create(c))
        resCreate := queue.ResBodyCreate{}
        bodyBytes, _ := ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resCreate, resCreate.Ticks);

        reqLogon := queue.ReqBodyLogon{}
        reqLogon.PrivateCode = resCreate.PrivateCode
        reqLogon.Ticks = 1592619000
        base64Encoded = defs.Encode(reqLogon, reqLogon.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	c = e.NewContext(req, rec)
	assert.NoError(t, queue.Logon(c))

	reqUpgrade := vendor.ReqBodyUpgrade{}
	reqUpgrade.Name = "vendor sample"
	reqUpgrade.Caption = "caption sample"
	reqUpgrade.Ticks = 1592619000
	base64Encoded = defs.Encode(reqUpgrade, reqUpgrade.Ticks)
	urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/vendor/upgrade", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx := &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.Upgrade(authCtx))
        resUpgrade := vendor.ResBodyUpgrade{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resUpgrade, resUpgrade.Ticks);

	// new queue require admit
        reqInitQueue := vendor.ReqBodyInitQueue{}
        reqInitQueue.RequireAdmit = true
        reqInitQueue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqInitQueue, reqInitQueue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/vendor/queue/new", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.InitQueue(authCtx))
        resInitQueue := vendor.ResBodyInitQueue{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resInitQueue, resInitQueue.Ticks);

        reqEnqueue := queue.ReqBodyEnqueue{}
        reqEnqueue.VendorCode = resUpgrade.VendorCode
        reqEnqueue.QueueCode = resInitQueue.QueueCode
        reqEnqueue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqEnqueue, reqEnqueue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/queue", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, queue.Enqueue(authCtx))
        resEnqueue := queue.ResBodyEnqueue{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resEnqueue, resEnqueue.Ticks);

	// show queue
        vendorCodeUrlUnsafed := resUpgrade.VendorCode
        queueCodeUrlUnsafed := resInitQueue.QueueCode
        r := strings.NewReplacer("=", "-", "/", "_", "+", ".")
        vendorCodeUrlSafed := r.Replace(vendorCodeUrlUnsafed)
        queueCodeUrlSafed := r.Replace(queueCodeUrlUnsafed)
	req = httptest.NewRequest(http.MethodGet, "/on/queue/"+vendorCodeUrlSafed+"/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/queue/:vendor_code/:queue_code")
	c.SetParamNames("vendor_code", "queue_code")
	c.SetParamValues(vendorCodeUrlSafed, queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, queue.ShowQueue(authCtx))

	// show queue vendor
	req = httptest.NewRequest(http.MethodGet, "/on/vendor/queue/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/queue/:queue_code")
	c.SetParamNames("queue_code")
	c.SetParamValues(queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.ShowQueue(authCtx))

	// manage queue vendor 
	req = httptest.NewRequest(http.MethodGet, "/on/vendor/manage/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/manage/:queue_code")
	c.SetParamNames("queue_code")
	c.SetParamValues(queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.Manage(authCtx))

	// polite dequeue by vendor
        reqDequeue := vendor.ReqBodyDequeue{}
	reqDequeue.Force = false
        reqDequeue.KeyCodePrefix = resEnqueue.KeyCodePrefix
        reqDequeue.KeyCodeSuffix = resEnqueue.KeyCodeSuffix
        reqDequeue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqDequeue, reqDequeue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/vendor/dequeue", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.Dequeue(authCtx))

	assert.NoError(t, priv.DropVendor(authCtx))
	assert.NoError(t, db.Teardown())
}

// Enqueue test require admit force dequeue
func TestEnqueueRequireAdmitForce(t *testing.T) {
	defs.ServicePrefix = "gotest"
	assert.NoError(t, db.Setup())
	assert.NoError(t, db.Conns.Init())
	assert.NoError(t, db.OpConns.Init())
	e := echo.New()
	route.Init(e)
	e.Logger.SetLevel(log.DEBUG)

	reqBody := queue.ReqBodyCreate{}
	reqBody.IdentifierType = 0
	reqBody.Identifier = "57ea5c1f17211a2c384a05030a88fcace73d9d92bd1c714da5c68ede09847304"
	reqBody.Seed = "9c463571a92614f5ed8ff55c249e7b8c458860e030284d2b5bcc7a529ac58741"
	reqBody.Ticks = 1592619000
	base64Encoded := defs.Encode(reqBody, reqBody.Ticks)
	urlEncoded := url.QueryEscape(base64Encoded)
	req := httptest.NewRequest(http.MethodPost, "/new", strings.NewReader(urlEncoded))
	rec := httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	c := e.NewContext(req, rec)

	assert.NoError(t, queue.Create(c))
        resCreate := queue.ResBodyCreate{}
        bodyBytes, _ := ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resCreate, resCreate.Ticks);

        reqLogon := queue.ReqBodyLogon{}
        reqLogon.PrivateCode = resCreate.PrivateCode
        reqLogon.Ticks = 1592619000
        base64Encoded = defs.Encode(reqLogon, reqLogon.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	c = e.NewContext(req, rec)
	assert.NoError(t, queue.Logon(c))

	reqUpgrade := vendor.ReqBodyUpgrade{}
	reqUpgrade.Name = "vendor sample"
	reqUpgrade.Caption = "caption sample"
	reqUpgrade.Ticks = 1592619000
	base64Encoded = defs.Encode(reqUpgrade, reqUpgrade.Ticks)
	urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/vendor/upgrade", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx := &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.Upgrade(authCtx))
        resUpgrade := vendor.ResBodyUpgrade{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resUpgrade, resUpgrade.Ticks);

	// new queue require admit
        reqInitQueue := vendor.ReqBodyInitQueue{}
        reqInitQueue.RequireAdmit = true
        reqInitQueue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqInitQueue, reqInitQueue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/vendor/queue/new", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.InitQueue(authCtx))
        resInitQueue := vendor.ResBodyInitQueue{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resInitQueue, resInitQueue.Ticks);

        reqEnqueue := queue.ReqBodyEnqueue{}
        reqEnqueue.VendorCode = resUpgrade.VendorCode
        reqEnqueue.QueueCode = resInitQueue.QueueCode
        reqEnqueue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqEnqueue, reqEnqueue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/queue", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, queue.Enqueue(authCtx))
        resEnqueue := queue.ResBodyEnqueue{}
        bodyBytes, _ = ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resEnqueue, resEnqueue.Ticks);

	// show queue
        vendorCodeUrlUnsafed := resUpgrade.VendorCode
        queueCodeUrlUnsafed := resInitQueue.QueueCode
        r := strings.NewReplacer("=", "-", "/", "_", "+", ".")
        vendorCodeUrlSafed := r.Replace(vendorCodeUrlUnsafed)
        queueCodeUrlSafed := r.Replace(queueCodeUrlUnsafed)
	req = httptest.NewRequest(http.MethodGet, "/on/queue/"+vendorCodeUrlSafed+"/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/queue/:vendor_code/:queue_code")
	c.SetParamNames("vendor_code", "queue_code")
	c.SetParamValues(vendorCodeUrlSafed, queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, queue.ShowQueue(authCtx))

	// show queue vendor
	req = httptest.NewRequest(http.MethodGet, "/on/vendor/queue/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/queue/:queue_code")
	c.SetParamNames("queue_code")
	c.SetParamValues(queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.ShowQueue(authCtx))

	// manage queue vendor 
	req = httptest.NewRequest(http.MethodGet, "/on/vendor/manage/"+queueCodeUrlSafed, nil)
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	c.SetPath("/on/manage/:queue_code")
	c.SetParamNames("queue_code")
	c.SetParamValues(queueCodeUrlSafed)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.Manage(authCtx))

	// force dequeue by vendor
        reqDequeue := vendor.ReqBodyDequeue{}
	reqDequeue.Force = true
        reqDequeue.KeyCodePrefix = resEnqueue.KeyCodePrefix
        reqDequeue.KeyCodeSuffix = ""
        reqDequeue.Ticks = 1592619000
        base64Encoded = defs.Encode(reqDequeue, reqDequeue.Ticks)
        urlEncoded = url.QueryEscape(base64Encoded)
	req = httptest.NewRequest(http.MethodPost, "/on/vendor/dequeue", strings.NewReader(urlEncoded))
	rec = httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	req.Header.Set("Session", resCreate.SessionId)
	c = e.NewContext(req, rec)
	authCtx = &defs.AuthContext{ c, 1 }
	assert.NoError(t, vendor.Dequeue(authCtx))

	assert.NoError(t, priv.DropVendor(authCtx))
	assert.NoError(t, db.Teardown())
}
