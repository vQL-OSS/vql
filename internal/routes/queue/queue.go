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
package queue

import (
	"database/sql"
	"encoding/base64"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	//"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vql/internal/db"
	"vql/internal/defs"
)

// Create user request body struct
type ReqBodyCreate struct {
	IdentifierType byte   `json:IdentifierType`
	Identifier     string `json:Identifier`
	Seed           string `json:Seed`
	defs.RequestBodyBase
}

// Create user response body struct
type ResBodyCreate struct {
	PrivateCode    string `json:PrivateCode`
	SessionId      string `json:SessionId`
	SessionPrivate string `json:SessionPrivate`
	defs.ResponseBodyBase
}

// Logon user request body struct
type ReqBodyLogon struct {
	PrivateCode string `json:PrivateCode`
	defs.RequestBodyBase
}

// Logon user response body struct
type ResBodyLogon struct {
	SessionId      string `json:SessionId`
	SessionPrivate string `json:SessionPrivate`
	defs.ResponseBodyBase
}

// Enqueue request body struct
type ReqBodyEnqueue struct {
	VendorCode string `json:VendorCode`
	QueueCode  string `json:QueueCode`
	defs.RequestBodyBase
}

// Enqueue response body struct
type ResBodyEnqueue struct {
	VendorName           string `json:VendorName`
	VendorCaption        string `json:VendorCaption`
	KeyCodePrefix        string `json:KeyCodePrefix`
	KeyCodeSuffix        string `json:KeyCodeSuffix`
	PersonsWaitingBefore int    `json:PersonsWaitingBefore`
	TotalWaiting         int    `json:TotalWaiting`
	defs.ResponseBodyBase
}

// Queue request body struct
type ReqBodyQueue struct {
	defs.RequestBodyBase
}

// Queue response body struct
type ResBodyQueue struct {
	PersonsWaitingBefore int `json:PersonsWaitingBefore`
	TotalWaiting         int `json:TotalWaiting`
	Status               int `json:Status`
	defs.ResponseBodyBase
}

// creates user account
func Create(c echo.Context) error {
	var err error
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := ReqBodyCreate{}
	response := ResBodyCreate{}
	response.ResponseCode = defs.ResponseOk
	response.SessionId = ""
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgEncodeInvalid, true, err))
	}

	// validate param
	c.Echo().Logger.Debugf("data: %v", request)
	platformType := c.Request().Header.Get("Platform")
	nonce := c.Request().Header.Get("Nonce")
	c.Echo().Logger.Debugf("nonce: %v", nonce)
	_, err = strconv.ParseInt(nonce, 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgNonceInvalid, true, err))
	}

	baseSeed := defs.ToHmacSha256(request.Identifier+platformType+strconv.FormatInt(request.Ticks, 10), defs.MagicKey)
	verifySeed := defs.ToHmacSha256(baseSeed+nonce, defs.MagicKey)
	c.Echo().Logger.Debug("seed : verifySeed -> %s : %s", request.Seed, verifySeed)
	if verifySeed != request.Seed {
		err = errors.New("failed verify seed")
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgSeedInvalid, true, err))
	}
	c.Echo().Logger.Debug("success verify seed")

	privateCode, err := defs.NewPrivateCode()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, err))
	}
	sessionId, err := defs.NewSession(string(privateCode[:]))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, err))
	}
	sessionPrivate, err := defs.NewSessionPrivate()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	// save create vendor master tables
	master := db.Conns.Master()
	var tx *sqlx.Tx
	var result sql.Result
	if tx, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	// already exists? -> logon or recover response
	// TODO seed exists check.
	// TODO auth exists check.
	if result, err = db.TxPreparexExec(tx, `insert into domain (
		service_code, vendor_code, shard, delete_flag, create_at, update_at
	) values (
		?, '', ?, 0, utc_timestamp(), utc_timestamp()
	)`, defs.ServiceCode, -1); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	signedId, err := result.LastInsertId()
	vendorId := uint64(signedId)

	if result, err = db.TxPreparexExec(tx, `insert into auth (
		id, identifier_type, platform_type, identifier, seed, secret,
		ticks, private_code, account_type, session_id, session_private, session_footprint, 
		delete_flag, create_at, update_at
	) values (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, utc_timestamp(), 0, utc_timestamp(), utc_timestamp()
	)`, vendorId, request.IdentifierType, platformType, request.Identifier, request.Seed, "", request.Ticks, privateCode, defs.NormalUser, sessionId, sessionPrivate); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("created")
	response.PrivateCode = base64.StdEncoding.EncodeToString(privateCode)
	response.SessionId = base64.StdEncoding.EncodeToString(sessionId)
	response.SessionPrivate = base64.StdEncoding.EncodeToString(sessionPrivate)
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Logon for keycode in queue
func Logon(c echo.Context) error {
	var err error
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := ReqBodyLogon{}
	response := ResBodyLogon{}
	response.ResponseCode = defs.ResponseOk
	response.SessionId = ""
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgEncodeInvalid, true, err))
	}
	decodedPrivateCode, err := base64.StdEncoding.DecodeString(request.PrivateCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	master := db.Conns.Master()
	var tx *sqlx.Tx
	if tx, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	var count int
	if err = db.TxPreparexGet(tx, "select count(1) from auth where to_base64(private_code) = ? ", &count, request.PrivateCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if count == 0 {
		err = errors.New("failed, private code not found. " + request.PrivateCode)
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(err, tx)))
	} else if count > 1 {
		err = errors.New("failed, invalid private code. " + request.PrivateCode)
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
	}

	sessionId, err := defs.NewSession(string(decodedPrivateCode[:]))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, db.RollbackResolve(err, tx)))
	}
	sessionPrivate, err := defs.NewSessionPrivate()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, db.RollbackResolve(err, tx)))
	}

	if _, err = db.TxPreparexExec(tx, `update auth set session_id = ?, session_private = ?, session_footprint = utc_timestamp(), update_at = utc_timestamp()
	where to_base64(private_code) = ?`,
		sessionId, sessionPrivate, request.PrivateCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("created")
	response.SessionId = base64.StdEncoding.EncodeToString(sessionId)
	response.SessionPrivate = base64.StdEncoding.EncodeToString(sessionPrivate)
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Add keycode in queue
func Enqueue(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := ReqBodyEnqueue{}
	response := ResBodyEnqueue{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgEncodeInvalid, true, err))
	}

	master := db.Conns.Master()
	var vendorId uint64
	c.Echo().Logger.Debugf("vendor code: %s", request.VendorCode)
	c.Echo().Logger.Debugf("queue code: %s", request.QueueCode)
	if err = db.PreparexGet(master, "select id from domain where to_base64(vendor_code) = ?", &vendorId, request.VendorCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgShardConnectFailed, true, err))
	}

	var tx *sqlx.Tx
	if tx, err = shard.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	summaryResult := struct {
		VendorName    string `db:"name"`
		VendorCaption string `db:"caption"`
	}{"", ""}
	queueResult := struct {
		Id            uint64
		KeyCodePrefix string `db:"keycode_prefix"`
		KeyCodeSuffix string `db:"keycode_suffix"`
	}{0, "", ""}
	var beforePerson int
	var total int
	var count int
	if err = db.TxPreparexGet(tx, `select count(1) from summary_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and delete_flag = 0`,
		&count, request.QueueCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if count == 0 {
		err = errors.New("failed, queue code not found. " + request.QueueCode)
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueueCodeNotfound, true, db.RollbackResolve(err, tx)))
	}

	if err = db.TxPreparexGet(tx, `select name, caption from summary_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and delete_flag = 0`,
		&summaryResult, request.QueueCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}


	if _, err = db.TxPreparexExec(tx, `insert into queue_`+db.ToSuffix(vendorId)+` (
		queue_code, uid, keycode_prefix, keycode_suffix, mail_addr, mail_count,
		push_type, push_count, status, delete_flag, create_at, update_at
	) values (
		from_base64(?), ?, cast(nextseq_`+db.ToSuffix(vendorId)+`("NUM") as char), "suffix_test", "", 0, 0, 0, 1, 0, utc_timestamp(), utc_timestamp()
	)`, request.QueueCode, authCtx.Uid); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = db.TxPreparexGet(tx, `select id, keycode_prefix, keycode_suffix from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and uid = ? and status = 1 and delete_flag = 0  limit 1`,
		&queueResult, request.QueueCode, authCtx.Uid); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = db.TxPreparexGet(tx, `select count(1) from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and id < ? and status = 1 and delete_flag = 0`,
		&beforePerson, request.QueueCode, queueResult.Id); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = db.TxPreparexGet(tx, `select count(1) from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and status = 1 and delete_flag = 0`,
		&total, request.QueueCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err := tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("enqueued")
	response.VendorName = summaryResult.VendorName
	response.VendorCaption = summaryResult.VendorCaption
	response.KeyCodePrefix = queueResult.KeyCodePrefix
	response.KeyCodeSuffix = queueResult.KeyCodeSuffix
	response.PersonsWaitingBefore = beforePerson
	response.TotalWaiting = total
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

type ShowQueueResult struct {
	Id     uint64
	Status int
}

// ShowQueue keycode list in queue
func ShowQueue(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	response := ResBodyQueue{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()
	_, err = strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}

	vendorCodeUrlSafed := c.Param("vendor_code")
	queueCodeUrlSafed := c.Param("queue_code")
	r := strings.NewReplacer("-", "=", "_", "/", ".", "+")
	vendorCode := r.Replace(vendorCodeUrlSafed)
	queueCode := r.Replace(queueCodeUrlSafed)

	if len(vendorCode) == 0 {
		err = errors.New("failed, vendor_code not found.")
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthNotFound, true, err))
	}
	if len(queueCode) == 0 {
		err = errors.New("failed, queue_code not found.")
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueueCodeNotfound, true, err))
	}

	master := db.Conns.Master()
	var vendorId uint64
	if err = db.PreparexGet(master, "select id from domain where to_base64(vendor_code) = ?",
		&vendorId, vendorCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	// create vendor shard tables.
	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgShardConnectFailed, true, err))
	}
	results := []ShowQueueResult{}
	var beforePerson int
	var total int

	if err = db.PreparexSelect(shard, `select id, status from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and uid = ? and delete_flag = 0 limit 1`,
		&results, queueCode, authCtx.Uid); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	if len(results) == 0 {
		err = errors.New("failed, keycode not found.")
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgKeyCodeCodeNotfound, true, err))
	}

	response.Status = results[0].Status
	if results[0].Status == 1 {
		if err = db.PreparexGet(shard, `select count(1) from queue_`+db.ToSuffix(vendorId)+
			` where to_base64(queue_code) = ? and id < ? and status = 1 and delete_flag = 0`,
			&beforePerson, queueCode, results[0].Id); err != nil {
			return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
		}
		if err = db.PreparexGet(shard, `select count(1) from queue_`+db.ToSuffix(vendorId)+
			` where to_base64(queue_code) = ? and status = 1 and delete_flag = 0`,
			&total, queueCode); err != nil {
			return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
		}
		response.PersonsWaitingBefore = beforePerson
		response.TotalWaiting = total
	}

	c.Echo().Logger.Debug("show queue")
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Dequeue vendor user request body struct
type ReqBodyDequeue struct {
	KeyCodePrefix string `json:"KeyCodePrefix"`
	KeyCodeSuffix string `json:"KeyCodeSuffix"`
	defs.RequestBodyBase
}

// Dequeue vendor user response body struct
type ResBodyDequeue struct {
	Updated bool
	defs.ResponseBodyBase
}

// Dequeue by user
func Dequeue(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := ReqBodyDequeue{}
	response := ResBodyDequeue{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgEncodeInvalid, true, err))
	}
	vendorId := authCtx.Uid

	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgShardConnectFailed, true, err))
	}
	var tx *sqlx.Tx
	var result sql.Result
	var updated int64
	if tx, err = shard.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}

	if result, err = db.TxPreparexExec(tx, `update queue_`+db.ToSuffix(vendorId)+
		` set status = ?, update_at = utc_timestamp() where uid = ? and keycode_prefix = ? and keycode_suffix = ?`,
		2, authCtx.Uid, request.KeyCodePrefix, request.KeyCodeSuffix); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}
	if updated, err = result.RowsAffected(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}
	if updated != 1 {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserDequeueFailed, true, db.RollbackResolve(err, tx)))
	}
	if err = tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("dequeue")
	response.Updated = updated == 1
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}
