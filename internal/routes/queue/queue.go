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
	"time"
	"vql/internal/db"
	"vql/internal/defs"
)

// Create user request body struct
type RequestBodyCreate struct {
	IdentifierType byte   `json:IdentifierType`
	Identifier     string `json:Identifier`
	Seed           string `json:Seed`
	defs.RequestBodyBase
}

// Create user response body struct
type ResponseBodyCreate struct {
	PrivateCode string `json:PrivateCode`
	SessionId   string `json:SessionId`
	SessionPrivate string `json:SessionPrivate`
	defs.ResponseBodyBase
}

// Logon user request body struct
type RequestBodyLogon struct {
	PrivateCode string `json:PrivateCode`
	defs.RequestBodyBase
}

// Logon user response body struct
type ResponseBodyLogon struct {
	SessionId      string `json:SessionId`
	SessionPrivate string `json:SessionPrivate`
	defs.ResponseBodyBase
}

// Enqueue request body struct
type RequestBodyEnqueue struct {
	VendorCode string `json:VendorCode`
	QueueCode  string `json:QueueCode`
	defs.RequestBodyBase
}

// Enqueue response body struct
type ResponseBodyEnqueue struct {
	KeyCodePrefix        string `json:KeyCodePrefix`
	KeyCodeSuffix        string `json:KeyCodeSuffix`
	PersonsWaitingBefore int    `json:PersonsWaitingBefore`
	TotalWaiting         int    `json:TotalWaiting`
	defs.ResponseBodyBase
}

// Queue request body struct
type RequestBodyQueue struct {
	VendorCode    string `json:VendorCode`
	QueueCode     string `json:QueueCode`
	KeyCodePrefix string `json:KeyCodePrefix`
	KeyCodeSuffix string `json:KeyCodeSuffix`
	defs.RequestBodyBase
}

// Queue response body struct
type ResponseBodyQueue struct {
	PersonsWaitingBefore int `json:PersonsWaitingBefore`
	TotalWaiting         int `json:TotalWaiting`
	Status               int `json:Status`
	defs.ResponseBodyBase
}

// creates user account
func Create(c echo.Context) error {
	var err error
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := RequestBodyCreate{}
	response := ResponseBodyCreate{}
	response.ResponseCode = defs.ResponseOk
	response.SessionId = ""
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgEncodeInvalid, true, err))
	}

	// validate param
	c.Echo().Logger.Debug("data: %v", request)
	platformType := c.Request().Header.Get("Platform")
	nonce := c.Request().Header.Get("Nonce")
	c.Echo().Logger.Debug("nonce: %v", nonce)
	_, err = strconv.ParseInt(nonce, 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgNonceInvalid, true, err))
	}

	baseSeed := defs.ToHmacSha256(request.Identifier+platformType+strconv.FormatInt(request.Ticks, 10), defs.MagicKey)
	verifySeed := defs.ToHmacSha256(baseSeed+nonce, defs.MagicKey)
	c.Echo().Logger.Debug("seed : verifySeed -> %s : %s", request.Seed, verifySeed)
	if verifySeed != request.Seed {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgSeedInvalid, true, errors.New("failed verify seed")))
	}
	c.Echo().Logger.Debug("success verify seed")

	privateCode, err := defs.NewPrivateCode()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, err))
	}
	sessionId, err := defs.NewSession(string(privateCode[:]))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, err))
	}
	sessionPrivate, err := defs.NewSessionPrivate()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	// save create vendor master tables
	master := db.Conns.Master()
	var tx *sqlx.Tx
	var result sql.Result
	if tx, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	// already exists? -> logon or recover response
	// TODO seed exists check.
	// TODO auth exists check.
	if result, err = db.TxPreparexExec(tx, `insert into domain (
		service_code, vendor_code, shard, delete_flag, create_at, update_at
	) values (
		?, '', ?, 0, utc_timestamp(), utc_timestamp()
	)`, defs.ServiceCode, -1); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	signedId, err := result.LastInsertId()
	vendorId := uint64(signedId)

	if result, err = db.TxPreparexExec(tx, `insert into auth (
		id, identifier_type, platform_type, identifier, seed, secret,
		ticks, private_code, session_id, session_private, session_footprint, delete_flag,
		create_at, update_at
	) values (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, utc_timestamp(), 0, utc_timestamp(), utc_timestamp()
	)`, vendorId, request.IdentifierType, platformType, request.Identifier, request.Seed, "", request.Ticks, privateCode, sessionId, sessionPrivate); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
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
	request := RequestBodyLogon{}
	response := ResponseBodyLogon{}
	response.ResponseCode = defs.ResponseOk
	response.SessionId = ""
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgEncodeInvalid, true, err))
	}
	decodedPrivateCode, err := base64.StdEncoding.DecodeString(request.PrivateCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	master := db.Conns.Master()
	var tx *sqlx.Tx
	if tx, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	var count int
	if err = db.TxPreparexGet(tx, "select count(1) from auth where to_base64(private_code) = ? ", &count, request.PrivateCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
	}

	if count == 0 {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(errors.New("failed, private code not found. "+request.PrivateCode), tx)))
	} else if count > 1 {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(errors.New("failed, invalid private code. "+request.PrivateCode), tx)))
	}

	sessionId, err := defs.NewSession(string(decodedPrivateCode[:]))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, db.RollbackResolve(err, tx)))
	}
	sessionPrivate, err := defs.NewSessionPrivate()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, db.RollbackResolve(err, tx)))
	}

	if _, err = db.TxPreparexExec(tx, `update auth set session_id = ?, session_private = ?, session_footprint = utc_timestamp(), update_at = utc_timestamp()
	where to_base64(private_code) = ?`,
	sessionId, sessionPrivate, request.PrivateCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
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
	request := RequestBodyEnqueue{}
	response := ResponseBodyEnqueue{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgEncodeInvalid, true, err))
	}

	master := db.Conns.Master()
	var vendorId uint64
	if err = db.PreparexGet(master, "select id from domain where to_base64(vendor_code) = ?", &vendorId, request.VendorCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}

	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgShardConnectFailed, true, err))
	}

	var tx *sqlx.Tx
	if tx, err = shard.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	result := struct {
		Id            uint64
		KeyCodePrefix string `db:"keycode_prefix"`
		KeyCodeSuffix string `db:"keycode_suffix"`
	}{0, "", ""}
	var beforePerson int
	var total int
	if _, err = db.TxPreparexExec(tx, `insert into queue_` + db.ToSuffix(vendorId) + ` (
		queue_code, uid, keycode_prefix, keycode_suffix, mail_addr, mail_count,
		push_type, push_count, status, delete_flag, create_at, update_at
	) values (
		from_base64(?), ?, cast(nextseq_` + db.ToSuffix(vendorId) + `("NUM") as char), "suffix_test", "", 0, 0, 0, 1, 0, utc_timestamp(), utc_timestamp()
	)`, request.QueueCode, authCtx.Uid); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = db.TxPreparexGet(tx, `select id, keycode_prefix, keycode_suffix from queue_` + db.ToSuffix(vendorId) +
	` where to_base64(queue_code) = ? and uid = ? and status = 1 and delete_flag = 0  limit 1`,
	&result, request.QueueCode, authCtx.Uid); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = db.TxPreparexGet(tx, `select count(1) from queue_` + db.ToSuffix(vendorId) +
	` where to_base64(queue_code) = ? and id < ? and status = 1 and delete_flag = 0`,
	&beforePerson, request.QueueCode, result.Id); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = db.TxPreparexGet(tx, `select count(1) from queue_` + db.ToSuffix(vendorId) +
	` where to_base64(queue_code) = ? and status = 1 and delete_flag = 0`,
	&total, request.QueueCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
	}

	if err := tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("enqueued")
	response.KeyCodePrefix = result.KeyCodePrefix
	response.KeyCodeSuffix = result.KeyCodeSuffix
	response.PersonsWaitingBefore = beforePerson
	response.TotalWaiting = total
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Get keycode list in queue
func Get(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := RequestBodyQueue{}
	response := ResponseBodyQueue{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgEncodeInvalid, true, err))
	}

	master := db.Conns.Master()
	var vendorId uint64
	if err = db.PreparexGet(master, "select id from domain where to_base64(vendor_code) = ?",
	&vendorId, request.VendorCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}

	// create vendor shard tables.
	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgShardConnectFailed, true, err))
	}
	var tx *sqlx.Tx
	if tx, err = shard.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	result := struct {
		Id     uint64
		Status int
	}{0, 0}
	var beforePerson int
	var total int
	var count int

	if err = db.TxPreparexGet(tx, `select count(1) from queue_` + db.ToSuffix(vendorId) +
	` where to_base64(queue_code) = ? and uid = ? and keycode_prefix = ? and keycode_suffix = ? and delete_flag = 0  limit 1`,
	&count, request.QueueCode, authCtx.Uid, request.KeyCodePrefix, request.KeyCodeSuffix); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
	}

	if count == 0 {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(errors.New("failed, keycode not found."), tx)))
	}

	if err = db.TxPreparexGet(tx, `select id, status from queue_` + db.ToSuffix(vendorId) +
	` where to_base64(queue_code) = ? and uid = ? and keycode_prefix = ? and keycode_suffix = ? and delete_flag = 0  limit 1`,
	&result, request.QueueCode, authCtx.Uid, request.KeyCodePrefix, request.KeyCodeSuffix); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
	}

	response.Status = result.Status
	if result.Status == 1 {
		if err = db.TxPreparexGet(tx, `select count(1) from queue_` + db.ToSuffix(vendorId) +
		` where to_base64(queue_code) = ? and id < ? and status = 1 and delete_flag = 0`,
		&beforePerson, request.QueueCode, result.Id); err != nil {
			return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
		}
		if err = db.TxPreparexGet(tx, `select count(1) from queue_` + db.ToSuffix(vendorId) +
		` where to_base64(queue_code) = ? and status = 1 and delete_flag = 0`,
		&total, request.QueueCode); err != nil {
			return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx)))
		}
		response.PersonsWaitingBefore = beforePerson
		response.TotalWaiting = total
	}

	if err := tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("fetch queue info")
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Update keycode in queue
func Update(c echo.Context) error {
	return c.String(http.StatusOK, "queue")
}
