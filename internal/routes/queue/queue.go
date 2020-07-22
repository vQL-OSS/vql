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
	"log"
	"net/http"
	"strconv"
	"time"
	"vql/internal/db"
	"vql/internal/defs"
)

// middleware function just to output message
func Middleware(name string) echo.MiddlewareFunc {
        return func(next echo.HandlerFunc) echo.HandlerFunc {
                return func(c echo.Context) error {
                        master := db.Conns.Master()
                        var count int
                        if err := master.Get(&count, "select count(1) from auth where session_id = ?", "Session"); err != nil {
                                log.Print(err)
                        }
                        if count == 0 {
                                return c.String(http.StatusInternalServerError, "")
                        }
                        err := next(c)
                        return err
                }
        }
}

// Create user request body struct
type RequestBodyCreate struct {
	IdentifierType byte   `json:IdentifierType`
	Identifier     string `json:Identifier`
	Seed           string `json:Seed`
	defs.MessageBodyBase
}

// Create user response body struct
type ResponseBodyCreate struct {
	PrivateCode string `json:PrivateCode`
	SessionId   string `json:SessionId`
	defs.ResponseBodyBase
}

// Logon user request body struct
type RequestBodyLogon struct {
	PrivateCode    string `json:PrivateCode`
	defs.MessageBodyBase
}

// Logon user response body struct
type ResponseBodyLogon struct {
	SessionId   string `json:SessionId`
	defs.ResponseBodyBase
}

// Enqueue request body struct
type RequestBodyEnqueue struct {
	SessionId   string `json:SessionId`
	VendorCode  string `json:VendorCode`
	QueueCode   string `json:QueueCode`
	defs.MessageBodyBase
}

// Enqueue response body struct
type ResponseBodyEnqueue struct {
	KeyCodePrefix   string `json:KeyCodePrefix`
	KeyCodeSuffix   string `json:KeyCodeSuffix`
	PersonsWaitingBefore  int `json:PersonsWaitingBefore`
	TotalWaiting  int `json:TotalWaiting`
	defs.ResponseBodyBase
}

// Queue request body struct
type RequestBodyQueue struct {
	KeyCodePrefix   string `json:KeyCodePrefix`
	KeyCodeSuffix   string `json:KeyCodeSuffix`
	defs.MessageBodyBase
}

// Queue response body struct
type ResponseBodyQueue struct {
	PersonsWaitingBefore  string `json:PersonsWaitingBefore`
	TotalWaiting  string `json:TotalWaiting`
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
	log.Printf("data: %v", request)
	platformType := c.Request().Header.Get("Platform")
	nonce := c.Request().Header.Get("Nonce")
	log.Printf("nonce: %v", nonce)
	_, err = strconv.ParseInt(nonce, 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgNonceInvalid, true, err))
	}

	baseSeed := defs.ToHmacSha256(request.Identifier+platformType+strconv.FormatInt(request.Ticks, 10), defs.MagicKey)
	verifySeed := defs.ToHmacSha256(baseSeed+nonce, defs.MagicKey)
	log.Printf("seed : verifySeed -> %s : %s", request.Seed, verifySeed)
	if verifySeed != request.Seed {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgSeedInvalid, true, errors.New("failed verify seed")))
	}
	log.Printf("success verify seed")
	privateCode, err := defs.NewPrivateCode()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, err))
	}
	sessionId, err := defs.NewSession()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	// save create vendor master tables
	master := db.Conns.Master()
	var tx1 *sqlx.Tx
	var result sql.Result
	if tx1, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, err))
	}
	// already exists? -> logon or recover response
	// TODO seed exists check.
	// TODO auth exists check.
	var stmt *sqlx.Stmt
	if stmt, err = tx1.Preparex("insert into domain (service_code, vendor_code, shard, delete_flag, create_at, update_at) values (?, '', ?, 0, utc_timestamp(), utc_timestamp())"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, err))
	}
	defer stmt.Close()
	if result, err = stmt.Exec(defs.ServiceCode, -1); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	signedId, err := result.LastInsertId()
	vendorId := uint64(signedId)

	if stmt, err = tx1.Preparex("insert into auth (id, identifier_type, platform_type, identifier, seed, secret, ticks, private_code, session_id, session_footprint, delete_flag, create_at, update_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, utc_timestamp(), 0, utc_timestamp(), utc_timestamp())"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if result, err = stmt.Exec(vendorId, request.IdentifierType, platformType, request.Identifier, request.Seed, "", request.Ticks, privateCode, sessionId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if err = tx1.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx1)))
	}

	c.Echo().Logger.Debug("created")
	response.PrivateCode = base64.StdEncoding.EncodeToString(privateCode)
	response.SessionId = base64.StdEncoding.EncodeToString(sessionId)
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
	sessionId, err := defs.NewSession()
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgHashGenerateFailed, true, err))
	}

	master := db.Conns.Master()
	var tx1 *sqlx.Tx
	if tx1, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, err))
	}
	var stmt *sqlx.Stmt
	var count int
	if stmt, err = tx1.Preparex("select count(1) from auth where to_base64(private_code) = ? "); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if err = stmt.Get(&count, request.PrivateCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}

	if (count == 0) {
                return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(errors.New("failed, private code not found. >>" + request.PrivateCode + "<<"), tx1)))
        } else if (count > 1) {
                return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(errors.New("failed, invalid private code. " + request.PrivateCode), tx1)))
        }

	if stmt, err = tx1.Preparex("update auth set session_id = ?, session_footprint = utc_timestamp(), update_at = utc_timestamp() where to_base64(private_code) = ?"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if _, err = stmt.Exec(sessionId, request.PrivateCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if err = tx1.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx1)))
	}

	c.Echo().Logger.Debug("created")
	response.SessionId = base64.StdEncoding.EncodeToString(sessionId)
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Search for keycode in queue
func Search(c echo.Context) error {
	return c.String(http.StatusOK, "queue")
}

// Detail keycode status in queue
func Detail(c echo.Context) error {
	return c.String(http.StatusOK, "queue")
}

// Get keycode list in queue
func Get(c echo.Context) error {
	return c.String(http.StatusOK, "queue")
}

// Add keycode in queue
func Enqueue(c echo.Context) error {
	var err error
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
	var tx1 *sqlx.Tx
	if tx1, err = master.Beginx(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgTransactBeginFailed, true, err))
	}
	var stmt *sqlx.Stmt
	var count int
	var Uid uint64
	var vendorId uint64
	if stmt, err = tx1.Preparex("select count(1) from auth where to_base64(session_id) = ? and date_add(session_footprint, interval 45 minute) > utc_timestamp()"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if err = stmt.Get(&count, request.SessionId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}

	if (count == 0) {
                return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(errors.New("failed, session not found."), tx1)))
        } else if (count > 1) {
                return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(errors.New("failed, invalid session."), tx1)))
        }

	if stmt, err = tx1.Preparex("select id from auth where to_base64(session_id) = ? and date_add(session_footprint, interval 45 minute) > utc_timestamp()"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if err = stmt.Get(&Uid, request.SessionId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}

	if stmt, err = tx1.Preparex("update auth set session_footprint = utc_timestamp() where to_base64(session_id) = ?"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if _, err = stmt.Exec(request.SessionId); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}

	if stmt, err = tx1.Preparex("select id from domain where to_base64(vendor_code) = ?"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}
	if err = stmt.Get(&vendorId, request.VendorCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx1)))
	}

	if err = tx1.Commit(); err != nil {
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
	result := struct {
		Id uint64
		KeycodePrefix string `db:"keycode_prefix"`
		KeycodeSuffix string `db:"keycode_suffix"`
	}{ 0, "", "", }
	var beforePerson int
	var total int
	stmt, err = tx2.Preparex(`insert into queue_` + db.ToSuffix(vendorId) + ` (
		queue_code, uid, keycode_prefix, keycode_suffix, mail_addr, mail_count, push_type, push_count, status, delete_flag, create_at, update_at
	) values (
		?, ?, cast(nextseq_` + db.ToSuffix(vendorId) +`("NUM") as char), "suffix_test", "", 0, 0, 0, 1, 0, utc_timestamp(), utc_timestamp()
	)`)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	_, err = stmt.Exec(request.QueueCode, Uid)
	if err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgQueryExecuteFailed, true, db.RollbackResolve(err, tx2)))
	}
	if stmt, err = tx2.Preparex("select id, keycode_prefix, keycode_suffix from queue_" + db.ToSuffix(vendorId) + " where queue_code = ? and uid = ? and status = 1 and delete_flag = 0  limit 1"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	if err = stmt.Get(&result, request.QueueCode, Uid); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}

	if stmt, err = tx2.Preparex("select count(1) from queue_" + db.ToSuffix(vendorId) + " where queue_code = ? and id < ? and status = 1 and delete_flag = 0"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	if err = stmt.Get(&beforePerson, request.QueueCode, result.Id); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}

	if stmt, err = tx2.Preparex("select count(1) from queue_" + db.ToSuffix(vendorId) + " where queue_code = ? and status = 1 and delete_flag = 0 limit 1"); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}
	if err = stmt.Get(&total, request.QueueCode); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgPreparedStatementFailed, true, db.RollbackResolve(err, tx2)))
	}

	if err := tx2.Commit(); err != nil {
		return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgCommitFailed, true, db.RollbackResolve(err, tx2)))
	}

	c.Echo().Logger.Debug("enqueued")
	response.KeyCodePrefix = result.KeycodePrefix
	response.KeyCodeSuffix = result.KeycodeSuffix
	response.PersonsWaitingBefore = beforePerson
	response.TotalWaiting = total
	return c.String(http.StatusOK, defs.Encode(response, response.Ticks))
}

// Update keycode in queue
func Update(c echo.Context) error {
	return c.String(http.StatusOK, "queue")
}
