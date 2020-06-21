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

package main

import (
	"flag"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"os"
	"os/signal"
	"log"
	"vql/internal/defs"
	"vql/internal/db"
	"vql/internal/routes/priv"
	"vql/internal/routes/queue"
	"vql/internal/routes/vendor"
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
	var err error

	defs.InitRand(true)
	err = db.Conns.Init()
	if err != nil {
		log.Fatal(err)
	}
	err = db.OpConns.Init()
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/vendor/new", vendor.Create)
	e.POST("/vendor/on/:id", vendor.Logon)
	e.PUT("/vendor/on/:id", vendor.Update)
	e.GET("/vendor/on/:id", vendor.Detail)
	e.POST("/vendor/on/auth/:id", vendor.AddAuth)
	e.PUT("/vendor/on/auth/:id", vendor.UpdateAuth)
	e.POST("/vendor/on/queue/:id", vendor.InitializeQueue)
	e.GET("/vendor/on/queue/:id", vendor.ShowQueue)
	e.PUT("/vendor/on/queue/:id", vendor.UpdateQueue)
	e.DELETE("/vendor/on/:id", vendor.Purge)
	e.POST("/vendor/off/:id", vendor.Logoff)

	e.GET("/queue/search/:id", queue.Search)
	e.GET("/queue/detail/:id", queue.Detail)
	e.GET("/queue/:id", queue.Get)
	e.GET("/queue/:id", queue.Add)
	e.GET("/queue/:id", queue.Update)

	e.DELETE("/priv/vendor", priv.DropVendor)

	e.Logger.Fatal(e.Start(":7000"))

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

}
