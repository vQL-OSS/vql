/* Copyright (c) 2018 FurtherSystem Co.,Ltd. All rights reserved.

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1335  USA */

package main

import (
	"flag"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"os"
	"os/signal"
	"vql/internal/routes/manage"
	"vql/internal/routes/queue"
)

var (
	standbyMode int
	repMode     bool
	logDir      string
)

func param() {
	flag.IntVar(&standbyMode, "standbymode", -1, "0=allcold, 1<standbymode is pre wake room, -1=allhot")
	flag.BoolVar(&repMode, "repmode", false, "replay mode ... false=off, true=on ")
	flag.StringVar(&logDir, "logdir", "/var/log/vqld/", "base log directory")
	flag.Parse()
}

func main() {
	param()
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/manage/queue", manage.SearchManage)
	e.GET("/queue", queue.SearchQueue)

	e.Logger.Fatal(e.Start(":7000"))

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
