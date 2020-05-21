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
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"vql/internal/defs"
)

// Create vendor user
func Create(c echo.Context) error {
	// TODO require SSO check is ok.
	// save uuid
	// TODO use connection pool
	// insert users
	_, err := sqlx.Open("mysql", "vql_user:password@tcp(127.0.0.1:3306)/"+defs.DBMasterName())
	if err != nil {
		log.Fatalln(err)
	}
	id := uint64(0)
	shard, err := sqlx.Open("mysql", "vql_user:password@tcp(127.0.0.1:3306)/"+defs.DBShardName(id))
	if err != nil {
		log.Fatalln(err)
	}
	// commit
	tx := shard.MustBegin()
	tx.MustExec(defs.CreateVendorQuery(id))
	tx.MustExec(defs.CreateQueueQuery(id))
	// generate keycodes
	tx.MustExec(defs.CreateKeyCodeQuery(id))
	tx.MustExec(defs.CreateAuthQuery(id))
	// commit
	tx.Commit()
	c.Echo().Logger.Debug("create")
	return c.String(http.StatusOK, "return master key and session_id here.")
}

// Logon vendor user
func Logon(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "return session_id here.")
}

// Update vendor user
func Update(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Detail vendor user
func Detail(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Add OAuth vendor user
func AddAuth(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "return master key and session_id here.")
}

// Update OAuth vendor user
func UpdateAuth(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "return master key and session_id here.")
}

// Initialize queue vendor user
func InitializeQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Show queue vendor user
func ShowQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Update queue vendor user
func UpdateQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Logoff vendor user
func Logoff(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Purge(logical remove) vendor user
func Purge(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}
