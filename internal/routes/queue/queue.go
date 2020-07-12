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
	"fmt"
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
			defer fmt.Printf("middleware-%s: defer\n", name)
			fmt.Printf("middleware-%s: before\n", name)
			err := next(c)
			fmt.Printf("middleware-%s: after\n", name)
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
func Add(c echo.Context) error {
	return c.String(http.StatusOK, "queue")
}

// Update keycode in queue
func Update(c echo.Context) error {
	return c.String(http.StatusOK, "queue")
}
