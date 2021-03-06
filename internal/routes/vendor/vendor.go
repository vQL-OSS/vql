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

// Vendor side web api package
package vendor

import (
	"database/sql"
	"encoding/base64"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vql/internal/db"
	"vql/internal/defs"
)

// Update vendor user request body struct
type ReqBodyUpdate struct {
	Name             string `json:Name`
	Caption          string `json:Caption`
	RequireInitQueue bool   `json:"RequireInitQueue"`
	RequireAdmit     bool   `json:"RequireAdmit"`
	defs.RequestBodyBase
}

// Update vendor user response body struct
type ResBodyUpdate struct {
	VendorCode string `json:VendorCode`
	QueueCode  string `json:"QueueCode"`
	defs.ResponseBodyBase
}

// Upgrade vendor
func Upgrade(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := ReqBodyUpdate{}
	response := ResBodyUpdate{}
	response.ResponseCode = defs.ResponseOk
	response.VendorCode = ""
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgEncodeInvalid, true, err))
	}

	// validate param
	c.Logger().Debugf("data: %v", request)
	nonce := c.Request().Header.Get("Nonce")
	c.Logger().Debugf("nonce: %v", nonce)
	_, err = strconv.ParseInt(nonce, 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgNonceInvalid, true, err))
	}

	vendorCode, err := defs.NewVendorCode()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	// save create vendor master tables
	master := db.Conns.Master()
	var tx1 *sqlx.Tx
	if tx1, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx1)))
	}
	// already exists? -> logon or recover response
	// TODO seed exists check.
	// TODO auth exists check.
	vendorId := authCtx.Uid
	var count int
	if err = db.TxPreparexGet(tx1, "select count(1) from auth where id = ? ", &count, vendorId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx1)))
	}

	if count == 0 {
		err = errors.New("failed, account not found.")
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(err, tx1)))
	} else if count > 1 {
		err = errors.New("failed, invalid account.")
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx1)))
	}

	if _, err = db.TxPreparexExec(tx1, "update domain set vendor_code = ?, update_at = utc_timestamp() where id = ?", vendorCode, vendorId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx1)))
	}

	if err := tx1.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx1)))
	}

	// create vendor shard tables.
	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgShardConnectFailed, true, err))
	}
	var tx2 *sqlx.Tx
	if tx2, err = shard.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx2)))
	}
	if _, err = db.TxPreparexExec(tx2, db.CreateSummaryQuery(vendorId)); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	if _, err = db.TxPreparexExec(tx2, db.CreateQueueQuery(vendorId)); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	if _, err = db.TxPreparexExec(tx2, db.CreateKeyCodeQuery(vendorId)); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	if _, err = db.TxPreparexExec(tx2, `insert into summary_`+db.ToSuffix(vendorId)+` (
		id, queue_code, reset_count, name, caption, require_admit, maintenance, delete_flag, create_at, update_at
	) values (
		?, '', 0, ?, ?, 0, 0, 0, utc_timestamp(), utc_timestamp()
	)`, 1, request.Name, request.Caption); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}

	if _, err = db.TxPreparexExec(tx2, db.CreateSequenceQuery(vendorId)); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}

	if _, err = db.TxPreparexExec(tx2, db.NewSequenceQuery(vendorId), "NUM", 0, 1); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}

	if _, err = tx2.Query(db.CreateFuncCurrSeqQuery(vendorId)); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}

	if _, err = tx2.Query(db.CreateFuncNextSeqQuery(vendorId)); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}

	if _, err = tx2.Query(db.CreateFuncUpdateSeqQuery(vendorId)); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}

	encodedQueueCode := ""
	if encodedQueueCode, err = InitQueue(c, tx2, vendorId, request.RequireAdmit, response, false); err != nil {
		return err
	}

	if err := tx2.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx2)))
	}

	// update shard = -1 -> shard = proper_shard_num
	assigned_shard := db.GetShardNum(vendorId)
	var tx3 *sqlx.Tx
	if tx3, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx3)))
	}
	if _, err = db.TxPreparexExec(tx3, "update domain set shard = ?, update_at = utc_timestamp() where id = ?", assigned_shard, vendorId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx3)))
	}
	if _, err = db.TxPreparexExec(tx3, "update auth set account_type = ?, update_at = utc_timestamp() where id = ?", defs.VendorUser, vendorId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx3)))
	}
	if err := tx3.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx3)))
	}

	c.Echo().Logger.Debugf("vendor code: %s", base64.StdEncoding.EncodeToString(vendorCode))
	c.Echo().Logger.Debugf("queue code: %s", encodedQueueCode)
	c.Echo().Logger.Debug("upgrade")
	response.VendorCode = base64.StdEncoding.EncodeToString(vendorCode)
	response.QueueCode = encodedQueueCode
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Initialize queue vendor user
func Update(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := ReqBodyUpdate{}
	response := ResBodyUpdate{}
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
	if tx, err = shard.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	if _, err = db.TxPreparexExec(tx, `update summary_`+db.ToSuffix(vendorId)+`
	set name = ?, caption = ?, update_at = utc_timestamp()
	where id = 1`, request.Name, request.Caption); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}
	encodedQueueCode := ""
	if request.RequireInitQueue {
		if encodedQueueCode, err = InitQueue(c, tx, vendorId, request.RequireAdmit, response, false); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("update vendor")
	response.QueueCode = encodedQueueCode
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Initialize queue vendor user
func InitQueue(c echo.Context, tx *sqlx.Tx, vendorId uint64, requireAdmit bool, response ResBodyUpdate, atFirst bool) (string, error) {
	var err error

	queueCode, err := defs.NewQueueCode()
	if err != nil {
		return "", c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	if !atFirst {
		db.TxPreparexExec(tx, `drop table queue_backup_`+db.ToSuffix(vendorId))
		if _, err = db.TxPreparexExec(tx, `create table queue_backup_`+db.ToSuffix(vendorId)+` like queue_`+db.ToSuffix(vendorId)); err != nil {
			return "", c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
		}
		if _, err = db.TxPreparexExec(tx, `truncate table queue_`+db.ToSuffix(vendorId)); err != nil {
			return "", c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
		}
	}
	if _, err = db.TxPreparexExec(tx, `update summary_`+db.ToSuffix(vendorId)+`
	set queue_code = ?, reset_count = cast(nextseq_`+db.ToSuffix(vendorId)+`("NUM") as char), require_admit = ?, update_at = utc_timestamp()
	where id = 1`, queueCode, requireAdmit); err != nil {
		return "", c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("init queue")
	return base64.StdEncoding.EncodeToString(queueCode), nil
}

// Dequeue vendor user request body struct

// Manage vendor user response body struct
type ResBodyManage struct {
	Name        string         `json:Name`
	Total       int            `json:"Total"`
	QueingTotal int            `json:"QueingTotal"`
	Rows        []ManageResult `json:"Rows"`
	defs.ResponseBodyBase
}

// Manage vendor db result struct
type ManageResult struct {
	KeyCodePrefix string `db:"keycode_prefix"`
	Status        int    `db:"status"`
}

// Manage vendor user
func Manage(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	response := ResBodyManage{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()
	_, err = strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}

	limitSize := 20
	queueCodeUrlSafed := c.Param("queue_code")
	r := strings.NewReplacer("-", "=", "_", "/", ".", "+")
	queueCode := r.Replace(queueCodeUrlSafed)
	pageStr := c.Param("page")
	if pageStr == "" {
		pageStr = "0"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgEncodeInvalid, true, err))
	}
	startIndex := page * limitSize

	if len(queueCode) == 0 {
		response.ResponseCode = defs.ResponseOkVendorRequireInitialize
		return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
	}

	master := db.Conns.Master()
	vendorId := authCtx.Uid
	var vendorCode string
	if err = db.PreparexGet(master, "select to_base64(vendor_code) from domain where id = ?",
		&vendorCode, authCtx.Uid); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	// create vendor shard tables.
	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgShardConnectFailed, true, err))
	}
	results := []ManageResult{}
	var total int
	var queingTotal int
	var name string

	if err = db.PreparexGet(shard, "select name from summary_"+db.ToSuffix(vendorId)+" where id = 1",
		&name); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}
	if err = db.PreparexGet(shard, `select count(1) from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and status != ? and delete_flag = 0`,
		&total, queueCode, defs.StatusCancel); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}
	if err = db.PreparexGet(shard, `select count(1) from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and status = ? and delete_flag = 0`,
		&queingTotal, queueCode, defs.StatusEnqueue); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}
	if err = db.PreparexSelect(shard, `select keycode_prefix, status from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and delete_flag = 0 limit ? offset ?`,
		&results, queueCode, limitSize, startIndex); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	c.Echo().Logger.Debug("manage")
	response.Name = name
	response.Total = total
	response.QueingTotal = queingTotal
	response.Rows = results
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Show Queue vendor user response body struct
type ResBodyShowQueue struct {
	Total       int               `json:"Total"`
	QueingTotal int               `json:"QueingTotal"`
	Rows        []ShowQueueResult `json:"Rows"`
	defs.ResponseBodyBase
}

// Show Queue vendor db result struct
type ShowQueueResult struct {
	KeyCodePrefix string `db:"keycode_prefix"`
	Status        int    `db:"status"`
}

// Show Queue vendor user
func ShowQueue(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	response := ResBodyShowQueue{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()
	_, err = strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}

	limitSize := 20
	queueCodeUrlSafed := c.Param("queue_code")
	r := strings.NewReplacer("-", "=", "_", "/", ".", "+")
	queueCode := r.Replace(queueCodeUrlSafed)
	pageStr := c.Param("page")
	if pageStr == "" {
		pageStr = "0"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgEncodeInvalid, true, err))
	}
	startIndex := page * limitSize

	if len(queueCode) == 0 {
		err = errors.New("failed, queue_code not found.")
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueueCodeNotfound, true, err))
	}

	master := db.Conns.Master()
	vendorId := authCtx.Uid
	var vendorCode string
	if err = db.PreparexGet(master, "select to_base64(vendor_code) from domain where id = ?",
		&vendorCode, authCtx.Uid); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	// create vendor shard tables.
	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgShardConnectFailed, true, err))
	}
	results := []ShowQueueResult{}
	var total int
	var queingTotal int

	if err = db.PreparexGet(shard, `select count(1) from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and delete_flag = 0`,
		&total, queueCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}
	if err = db.PreparexGet(shard, `select count(1) from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and status = ? and delete_flag = 0`,
		&queingTotal, queueCode, defs.StatusEnqueue); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}
	if err = db.PreparexSelect(shard, `select keycode_prefix, status from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and status = ? and delete_flag = 0 limit ? offset ?`,
		&results, queueCode, defs.StatusEnqueue, limitSize, startIndex); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	c.Echo().Logger.Debug("vendor show queue")
	response.Total = total
	response.QueingTotal = queingTotal
	response.Rows = results
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Manage vendor user response body struct
type ResBodyDetail struct {
	Name    string `json:"Name"`
	Caption string `json:"Caption"`
	defs.ResponseBodyBase
}

// Manage vendor db result struct
type DetailResult struct {
	Name    string `db:"name"`
	Caption string `db:"caption"`
}

// Get detail vendor user
func Detail(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	response := ResBodyDetail{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()
	_, err = strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTicksInvalid, true, err))
	}

	vendorId := authCtx.Uid
	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgShardConnectFailed, true, err))
	}
	result := DetailResult{}
	if err = db.PreparexGet(shard, "select name, caption from summary_"+db.ToSuffix(vendorId)+" where id = 1",
		&result); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	c.Echo().Logger.Debug("vendor detail")
	response.Name = result.Name
	response.Caption = result.Caption
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

type ReqBodyDequeue struct {
	Force         bool   `json:"Force"`
	KeyCodePrefix string `json:"KeyCodePrefix"`
	KeyCodeSuffix string `json:"KeyCodeSuffix"`
	defs.RequestBodyBase
}

// Dequeue vendor user response body struct
type ResBodyDequeue struct {
	Updated bool
	defs.ResponseBodyBase
}

// Dequeue by vendor user
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

	if request.Force {
		if result, err = db.TxPreparexExec(tx, `update queue_`+db.ToSuffix(vendorId)+
			` set status = ?, update_at = utc_timestamp() where keycode_prefix = ?`,
			defs.StatusDequeue, request.KeyCodePrefix); err != nil {
			return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
		}
	} else {
		if result, err = db.TxPreparexExec(tx, `update queue_`+db.ToSuffix(vendorId)+
			` set status = ?, update_at = utc_timestamp() where keycode_prefix = ? and keycode_suffix = ?`,
			defs.StatusDequeue, request.KeyCodePrefix, request.KeyCodeSuffix); err != nil {
			return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
		}
	}
	if updated, err = result.RowsAffected(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}
	if updated > 1 {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}
	if err = tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("vendor dequeue")
	response.Updated = updated == 1
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Enqueue dummy request body struct ... no use

// Enqueue dummy response body struct
type ResBodyEnqueueDummy struct {
	KeyCodePrefix string `json:KeyCodePrefix`
	KeyCodeSuffix string `json:KeyCodeSuffix`
	defs.ResponseBodyBase
}

// Enqueue dummy
func EnqueueDummy(c echo.Context) error {
	var err error
	authCtx := c.(*defs.AuthContext)
	response := ResBodyEnqueueDummy{}
	response.ResponseCode = defs.ResponseOk
	response.Ticks = time.Now().Unix()

	master := db.Conns.Master()
	vendorId := authCtx.Uid
	var vendorCodeBase64 string
	if err = db.PreparexGet(master, "select to_base64(vendor_code) from domain where id = ?", &vendorCodeBase64, vendorId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, err))
	}

	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgShardConnectFailed, true, err))
	}

	keyCodeSuffix, err := defs.NewKeyCodeSuffix()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	var tx *sqlx.Tx
	if tx, err = shard.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgTransactBeginFailed, true, db.RollbackResolve(err, tx)))
	}
	result := struct {
		Id            uint64
		KeyCodePrefix string `db:"keycode_prefix"`
		KeyCodeSuffix string `db:"keycode_suffix"`
	}{0, "", ""}
	var queueCodeBase64 string
	if err = db.TxPreparexGet(tx, `select to_base64(queue_code) from summary_`+db.ToSuffix(vendorId)+
		` where id = ? and delete_flag = 0`,
		&queueCodeBase64, vendorId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if _, err = db.TxPreparexExec(tx, `insert into queue_`+db.ToSuffix(vendorId)+` (
                queue_code, uid, keycode_prefix, keycode_suffix, mail_addr, mail_count,
                push_type, push_count, status, delete_flag, create_at, update_at
        ) values (
                from_base64(?), ?, cast(nextseq_`+db.ToSuffix(vendorId)+`("NUM") as char), ?, "", 0, 0, 0, ?, 0, utc_timestamp(), utc_timestamp()
        )`, queueCodeBase64, authCtx.Uid, keyCodeSuffix, defs.StatusEnqueue); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err = db.TxPreparexGet(tx, `select id, keycode_prefix, keycode_suffix from queue_`+db.ToSuffix(vendorId)+
		` where to_base64(queue_code) = ? and uid = ? and status = ? and delete_flag = 0  limit 1`,
		&result, queueCodeBase64, authCtx.Uid, defs.StatusEnqueue); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx)))
	}

	if err := tx.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx)))
	}

	c.Echo().Logger.Debug("dummy enqueued")
	response.KeyCodePrefix = result.KeyCodePrefix
	response.KeyCodeSuffix = result.KeyCodeSuffix
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Purge vendor user request body struct
type ReqBodyPurge struct{}

// Purge vendor user response body struct
type ResBodyPurge struct{}

// Purge(logical remove) vendor user
func Purge(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}
