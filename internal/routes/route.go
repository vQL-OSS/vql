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

package route

import (
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"vql/internal/db"
	"vql/internal/defs"
	"vql/internal/routes/priv"
	"vql/internal/routes/queue"
	"vql/internal/routes/vendor"
)

func Init(e *echo.Echo) {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/new", queue.Create)
	e.POST("/logon", queue.Logon)
	g := e.Group("/on")
	g.Use(AuthMiddleware())
	g.POST("/queue", queue.Enqueue)
	g.POST("/vendor/upgrade", vendor.Upgrade)
	e.DELETE("/priv/vendor", priv.DropVendor)
}

// middleware function just to output message
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			master := db.Conns.Master()
			sessionId := c.Request().Header.Get("Session")
			nonce := c.Request().Header.Get("Nonce")
			hash := c.Request().Header.Get("Hash")
			response := defs.ResponseBodyBase{}

			var tx *sqlx.Tx
			var count int
			var err error
			var sessionPrivate string
			if tx, err = master.Beginx(); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
			}
			if err = db.TxPreparexGet(tx, "select count(1) from auth where to_base64(session_id) = ? and date_add(session_footprint, interval "+defs.SessionTimeout+" minute) > utc_timestamp()", &count, sessionId); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(err, tx)))
			}

			if count == 0 {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(errors.New("failed, session not found. "+sessionId), tx)))
			} else if count > 1 {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(errors.New("failed, invalid session. "+sessionId), tx)))
			}

			if err = db.TxPreparexGet(tx, "select to_base64(session_private) from auth where to_base64(session_id) = ?", &sessionPrivate, sessionId); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(err, tx)))
			}

			// check validate hash
			verifyHash := defs.ToHmacSha256(sessionPrivate+nonce, defs.MagicKey)
			if hash != verifyHash {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(errors.New("failed, verify session. "+sessionId), tx)))
			}

			ac := &defs.AuthContext{ c, 0 }
			if err = db.TxPreparexGet(tx, "select id from auth where to_base64(session_id) = ?", &ac.Uid, sessionId); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(err, tx)))
			}

			if _, err = db.TxPreparexExec(tx, "update auth set session_footprint = utc_timestamp() where to_base64(session_id) = ?", sessionId); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
			}

			if err = tx.Commit(); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(&response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
			}
			return next(ac)
		}
	}
}
