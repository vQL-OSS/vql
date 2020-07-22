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
	"encoding/base64"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
	"vql/internal/db"
	"vql/internal/defs"
)

// Upgrade vendor user request body struct
type RequestBodyUpgrade struct {
	SessionId      string `json:SessionId`
	Name           string `json:Name`
	Caption        string `json:Caption`
	defs.MessageBodyBase
}

// Upgrade vendor user response body struct
type ResponseBodyUpgrade struct {
	VendorCode  string `json:VendorCode`
	defs.ResponseBodyBase
}

// Upgrade vendor
func Upgrade(c echo.Context) error {
	var err error
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	request := RequestBodyUpgrade{}
	response := ResponseBodyUpgrade{}
	response.ResponseCode = defs.ResponseOk
	response.VendorCode = ""
	response.Ticks = time.Now().Unix()
	ticks, err := strconv.ParseInt(c.Request().Header.Get("IV"), 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTicksInvalid, true, err))
	}
	if err = defs.Decode(bodyBytes, &request, ticks); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgEncodeInvalid, true, err))
	}

	// validate param
	log.Printf("data: %v", request)
	nonce := c.Request().Header.Get("Nonce")
	log.Printf("nonce: %v", nonce)
	_, err = strconv.ParseInt(nonce, 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgNonceInvalid, true, err))
	}

	vendorCode, err := defs.NewVendorCode()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	// save create vendor master tables
	master := db.Conns.Master()
	var tx1 *sqlx.Tx
	if tx1, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, err))
	}
	// already exists? -> logon or recover response
	// TODO seed exists check.
	// TODO auth exists check.
	var stmt *sqlx.Stmt
	var vendorId uint64
	var count int
	if stmt, err = tx1.Preparex("select count(1) from auth where to_base64(session_id) = ? "); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}
	defer stmt.Close()
	if err := stmt.Get(&count, request.SessionId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}

	if (count == 0) {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(errors.New("failed, account not found."), tx1)))
	} else if (count > 1) {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(errors.New("failed, invalid account."), tx1)))
	}

	if stmt, err = tx1.Preparex("select id from auth where to_base64(session_id) = ? limit 1"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}
	defer stmt.Close()
	if err := stmt.Get(&vendorId, request.SessionId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}

	if stmt, err = tx1.Preparex("update domain set vendor_code = ?, update_at = utc_timestamp() where id = ?"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}
	defer stmt.Close()
	if _, err = stmt.Exec(vendorCode, vendorId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if err := tx1.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx1)))
	}

	// create vendor shard tables.
	shard, err := db.Conns.Shard(vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgShardConnectFailed, true, err))
	}
	var tx2 *sqlx.Tx
	if tx2, err = shard.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, err))
	}
	stmt, err = tx2.Preparex(db.CreateSummaryQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}
	_, err = stmt.Exec()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	stmt, err = tx2.Preparex(db.CreateQueueQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = stmt.Exec()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	stmt, err = tx2.Preparex(db.CreateKeycodeQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = stmt.Exec()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	stmt, err = tx2.Preparex(db.CreateAuthQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = stmt.Exec()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	stmt, err = tx2.Preparex(`insert into summary_` + db.ToSuffix(vendorId) + ` (
		id, queue_code, reset_count, name, first_code,
		last_code, total_wait, total_in, total_out, maintenance,
		caption, delete_flag, create_at, update_at
	) values (
		?, '', 0, ?, '',
		'', 0, 0, 0, 0,
		?, 0, utc_timestamp(), utc_timestamp()
	)`)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = stmt.Exec(vendorId, request.Name, request.Caption)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}

	stmt, err = tx2.Preparex(db.CreateSequenceQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = stmt.Exec()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	stmt, err = tx2.Preparex(db.NewSequenceQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = stmt.Exec("NUM", 0, 1)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}

	_, err = tx2.Query(db.CreateFuncCurrSeqQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = tx2.Query(db.CreateFuncNextSeqQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = tx2.Query(db.CreateFuncUpdateSeqQuery(vendorId))
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}

	if err := tx2.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx2)))
	}

	// update shard = -1 -> shard = proper_shard_num
	assigned_shard := db.GetShardNum(vendorId)
	stmt, err = master.Preparex("update domain set shard = ?, update_at = utc_timestamp() where id = ?")
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}
	_, err = stmt.Exec(assigned_shard, vendorId)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, err))
	}
	c.Echo().Logger.Debug("created")
	response.VendorCode = base64.StdEncoding.EncodeToString(vendorCode)
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Logon vendor user request body struct
type ReqBodyLogon struct{}

// Logon vendor user response body struct
type ResBodyLogon struct{}

// Logon vendor user
func Logon(c echo.Context) error {
	// TODO require SSO check is ok.
	// create session id and update respond session_id
	return c.String(http.StatusOK, "return session_id here.")
}

// Update vendor user request body struct
type ReqBodyUpdate struct{}

// Update vendor user response body struct
type ResBodyUpdate struct{}

// Update vendor user
func Update(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Detail vendor user request body struct
type ReqBodyDetail struct{}

// Detail vendor user response body struct
type ResBodyDetail struct{}

// Detail vendor user
func Detail(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Add OAuth vendor user request body struct
type ReqBodyAddAuth struct{}

// Add OAuth vendor user response body struct
type ResBodyAddAuth struct{}

// Add OAuth vendor user
func AddAuth(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "return master key and session_id here.")
}

// Update OAuth vendor user request body struct
type ReqBodyUpdateAuth struct{}

// Update OAuth vendor user response body struct
type ResBodyUpdateAuth struct{}

// Update OAuth vendor user
func UpdateAuth(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "return master key and session_id here.")
}

// Initialize queue vendor user request body struct
type ReqBodyInitializeQueue struct{}

// Initialize queue vendor user response body struct
type ResBodyInitializeQueue struct{}

// Initialize queue vendor user
func InitializeQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	// generate keycodes here
	return c.String(http.StatusOK, "vendor")
}

// Show queue vendor user request body struct
type ReqBodyShowQueue struct{}

// Show queue vendor user response body struct
type ResBodyShowQueue struct{}

// Show queue vendor user
func ShowQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Update queue vendor user request body struct
type ReqBodyUpdateQueue struct{}

// Update queue vendor user response body struct
type ResBodyUpdateQueue struct{}

// Update queue vendor user
func UpdateQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Dequeue vendor user request body struct
type ReqBodyDequeue struct{}

// Dequeue vendor user response body struct
type ResBodyDequeue struct{}

// Dequeue(logical remove) vendor user
func Dequeue(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "")
}

// Logoff vendor user request body struct
type ReqBodyLogoff struct{}

// Logoff vendor user response body struct
type ResBodyLogoff struct{}

// Logoff vendor user
func Logoff(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
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
