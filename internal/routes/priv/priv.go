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
        //"github.com/jmoiron/sqlx"
	//"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"net/http"
	"vql/internal/db"
	"vql/internal/defs"
)

// Drop(physics remove) vendor
func DropVendor(c echo.Context) error {
	authCtx := c.(*defs.AuthContext)
	master := db.OpConns.Master()
	stmt, err := master.Preparex(`select * from domain where id = ?`)
	domain := db.Domain{}
	paramId := authCtx.Uid
	stmt.Exec(&domain, paramId)
	if err != nil {
		return err
	}
	defer stmt.Close()
	shard, err := db.OpConns.Shard(domain.Id)
	if err != nil {
		return err
	}
	tx, err := shard.Beginx()
	if err != nil {
		return err
	}
	stmt, err = tx.Preparex(db.DropSummaryQuery(domain.Id))
	if err != nil {
		return err
	}
	defer stmt.Close()
	stmt.Exec()
	stmt, err = tx.Preparex(db.DropQueueQuery(domain.Id))
	if err != nil {
		return err
	}
	defer stmt.Close()
	stmt.Exec()
	// generate keycodes
	stmt, err = tx.Preparex(db.DropKeyCodeQuery(domain.Id))
	if err != nil {
		return err
	}
	defer stmt.Close()
	stmt.Exec()
	stmt, err = tx.Preparex(db.DropAuthQuery(domain.Id))
	if err != nil {
		return err
	}
	defer stmt.Close()
	stmt.Exec()
	// commit
	tx.Commit()
	c.Echo().Logger.Debug("removed")
	return c.String(http.StatusOK, "return master key here.")

	c.Logger().Debug("removed")
	return c.String(http.StatusOK, "")
}
