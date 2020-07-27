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

// Create user test
func TestCreate(t *testing.T) {
	defs.ServicePrefix = "gotest"
	assert.NoError(t, db.Setup())
	assert.NoError(t, db.Conns.Init())
	assert.NoError(t, db.OpConns.Init())
	e := echo.New()
	route.Init(e)

	reqBody := queue.RequestBodyCreate{}
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
	assert.NoError(t, priv.DropVendor(c))
	assert.NoError(t, db.Teardown())
}

// Logon test
func TestLogon(t *testing.T) {
	defs.ServicePrefix = "gotest"
	assert.NoError(t, db.Setup())
	assert.NoError(t, db.Conns.Init())
	assert.NoError(t, db.OpConns.Init())
	e := echo.New()
	reqBody := queue.RequestBodyCreate{}
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
        resCreate := queue.ResponseBodyCreate{}
        bodyBytes, _ := ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resCreate, resCreate.Ticks);

        reqLogon := queue.RequestBodyLogon{}
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


	assert.NoError(t, priv.DropVendor(c))
	assert.NoError(t, db.Teardown())
}

// Enqueue test
func TestEnqueue(t *testing.T) {
	defs.ServicePrefix = "gotest"
	assert.NoError(t, db.Setup())
	assert.NoError(t, db.Conns.Init())
	assert.NoError(t, db.OpConns.Init())
	e := echo.New()
	route.Init(e)

	reqBody := queue.RequestBodyCreate{}
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
        resCreate := queue.ResponseBodyCreate{}
        bodyBytes, _ := ioutil.ReadAll(rec.Body)
        defs.Decode(bodyBytes, &resCreate, resCreate.Ticks);

        reqLogon := queue.RequestBodyLogon{}
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

	reqUpgrade := vendor.RequestBodyUpgrade{}
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
	assert.NoError(t, vendor.Upgrade(c))

        reqEnqueue := queue.RequestBodyEnqueue{}
        reqEnqueue.VendorCode = ""
        reqEnqueue.QueueCode = ""
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
	authCtx := &defs.AuthContext{ c, 1 }
	assert.NoError(t, queue.Enqueue(authCtx))

	assert.NoError(t, priv.DropVendor(c))
	assert.NoError(t, db.Teardown())
}

// Upgrade vendor user test
func TestUpgrade(t *testing.T) {
	defs.ServicePrefix = "gotest"
	assert.NoError(t, db.Setup())
	assert.NoError(t, db.Conns.Init())
	assert.NoError(t, db.OpConns.Init())
	e := echo.New()
	route.Init(e)

	reqCreate := queue.RequestBodyCreate{}
	reqCreate.IdentifierType = 0
	reqCreate.Identifier = "57ea5c1f17211a2c384a05030a88fcace73d9d92bd1c714da5c68ede09847304"
	reqCreate.Seed = "9c463571a92614f5ed8ff55c249e7b8c458860e030284d2b5bcc7a529ac58741"
	reqCreate.Ticks = 1592619000
	base64Encoded := defs.Encode(reqCreate, reqCreate.Ticks)
	urlEncoded := url.QueryEscape(base64Encoded)
	req := httptest.NewRequest(http.MethodPost, "/new", strings.NewReader(urlEncoded))
	rec := httptest.NewRecorder()
	req.Header.Set("User-Agent", "vQL-Client")
	req.Header.Set("Platform", "Windows")
	req.Header.Set("IV", "0")
	req.Header.Set("Nonce", "637295289927929882")
	c := e.NewContext(req, rec)

	assert.NoError(t, queue.Create(c))

	resCreate := queue.ResponseBodyCreate{}
	bodyBytes, _ := ioutil.ReadAll(rec.Body)
	defs.Decode(bodyBytes, &resCreate, resCreate.Ticks);

	reqUpgrade := vendor.RequestBodyUpgrade{}
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
	assert.NoError(t, priv.DropVendor(c))
	assert.NoError(t, db.Teardown())
}
