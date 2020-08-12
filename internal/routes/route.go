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
	g.GET("/queue/:vendor_code/:queue_code", queue.ShowQueue)
	g.POST("/dequeue", queue.Dequeue)
	g.POST("/vendor/upgrade", vendor.Upgrade)
	g.POST("/vendor/queue/new", vendor.InitQueue)
	g.POST("/vendor/queue/dummy", vendor.EnqueueDummy)
	g.GET("/vendor/manage/", vendor.Manage)
	g.GET("/vendor/manage/:queue_code/:page", vendor.Manage)
	g.GET("/vendor/queue/:queue_code/:page", vendor.ShowQueue)
	g.POST("/vendor/dequeue", vendor.Dequeue)
	g.DELETE("/priv/vendor", priv.DropVendor)
}

type AuthResult struct {
        Id     uint64
	SessionPrivate string `db:"session_private"`
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
			var err error
			results := []AuthResult{}
			ac := &defs.AuthContext{c, 0}
			if tx, err = master.Beginx(); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
			}
			if err = db.TxPreparexSelect(tx, `select id, to_base64(session_private) as session_private from auth where to_base64(session_id) = ? and date_add(session_footprint, interval `+
				defs.SessionTimeout+` minute) > utc_timestamp()`, &results, sessionId); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(err, tx)))
			}

			count := len(results)
			if count == 0 {
				err = errors.New("failed, session not found. " + sessionId)
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthNotFound, true, db.RollbackResolve(err, tx)))
			} else if count > 1 {
				err = errors.New("failed, invalid session. " + sessionId)
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
			}

			// check validate hash
			verifyHash := defs.ToHmacSha256(results[0].SessionPrivate+nonce, defs.MagicKey)
			if hash != verifyHash {
				err = errors.New("failed, verify session. " + sessionId)
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
			}

			ac.Uid = results[0].Id

			if _, err = db.TxPreparexExec(tx, "update auth set session_footprint = utc_timestamp() where to_base64(session_id) = ?", sessionId); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
			}

			if err = tx.Commit(); err != nil {
				return c.String(http.StatusInternalServerError, defs.ErrorDispose(c, &response, defs.ResponseNgUserAuthFailed, true, db.RollbackResolve(err, tx)))
			}
			return next(ac)
		}
	}
}
