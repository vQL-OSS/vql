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
	"github.com/labstack/echo/v4"
	"net/http"
	"fmt"
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
