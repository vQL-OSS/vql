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

package queue

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSearch(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/queue", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	assert.NoError(t, SearchQueue(c))
}
