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

// Privilege access web api package
package priv

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"vql/internal/defs"
)

// Drop(physics remove) vendor user
func DropVendor(c echo.Context) error {
	// TODO require SSO check is ok.
	// save uuid
	// TODO use connection pool
	_, err := sqlx.Open("mysql", "vql_opuser:password@tcp(127.0.0.1:3306)/"+defs.DBMasterName())
	if err != nil {
		log.Fatalln(err)
	}
	id := uint64(0)
	shard, err := sqlx.Open("mysql", "vql_opuser:password@tcp(127.0.0.1:3306)/"+defs.DBShardName(id))
	if err != nil {
		log.Fatalln(err)
	}
	// insert users
	// commit
	tx := shard.MustBegin()
	tx.MustExec(defs.DropVendorQuery(id))
	tx.MustExec(defs.DropQueueQuery(id))
	// generate keycodes
	tx.MustExec(defs.DropKeyCodeQuery(id))
	tx.MustExec(defs.DropAuthQuery(id))
	// commit
	tx.Commit()
	c.Echo().Logger.Debug("remove")
	return c.String(http.StatusOK, "return master key here.")
}
