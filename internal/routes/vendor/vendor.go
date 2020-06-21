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
	"database/sql"
	"log"
	"io/ioutil"
	"net/http"
	"vql/internal/db"
	"vql/internal/defs"
	"strconv"
	"errors"
)

// Create vendor user request body struct
type ReqBodyCreate struct {
	IdentifierType	byte	`json:IdentifierType`
	Identifier	string	`json:Identifier`
	Seed		string	`json:Seed`
	Name		string	`json:Name`
	Caption		string	`json:Caption`
	Ticks		int64	`json:Ticks`
}
// Create vendor user response body struct
type ResBodyCreate struct {
	ResponseCode	int	`json:ResponseCode`
	VendorId	uint64	`json:VendorId`
	PublicCode	string	`json:PublicCode`
	PrivateCode	string	`json:PrivateCode`
	SessionId	string	`json:SessionId`
	Ticks		int64	`json:Ticks`
}
// Create vendor
func Create(c echo.Context) error {
	var err error
	// TODO require SSO check is ok.
	bodyBytes, err := ioutil.ReadAll(c.Request().Body)
	reqBodyCreate := ReqBodyCreate{}
	if err = defs.Decode(bodyBytes, &reqBodyCreate); err != nil { return err }
	log.Printf("data: %v",reqBodyCreate)
	platformType := c.Request().Header.Get("Platform")
	nonce := c.Request().Header.Get("Nonce")
	// validate seed
	baseSeed := defs.ToHmacSha256(reqBodyCreate.Identifier + platformType + strconv.FormatInt(reqBodyCreate.Ticks, 10) , defs.MagicKey)
	verifySeed := defs.ToHmacSha256(baseSeed + nonce, defs.MagicKey)
	log.Printf("seed : verifySeed -> %s : %s", reqBodyCreate.Seed, verifySeed)
	if verifySeed != reqBodyCreate.Seed {
		return errors.New("failed verify seed")
	}
	log.Printf("success verify seed")
	// save uuid
	vendorCode, err := defs.NewVendorCode()
	if err != nil { return err }
	privateCode, err := defs.NewPrivateCode()
	if err != nil { return err }
	sessionId, err := defs.NewSession()
	if err != nil { return err }
	master := db.Conns.Master()

	var tx1 *sqlx.Tx
	var res sql.Result

	if tx1, err = master.Beginx(); err != nil { return err }
	// already exists? -> logon or recover response 
	var stmt *sqlx.Stmt
	if stmt, err = tx1.Preparex("insert into domain (service_code, vendor_code, shard, delete_flag, create_at, update_at) values (?, ?, ?, 0, utc_timestamp(), utc_timestamp())"); err != nil { return err }
	defer stmt.Close()
	if res, err = stmt.Exec(defs.ServiceCode, vendorCode, -1); err != nil {
		return db.RollbackResolve(err, tx1)
	}
	signedId, err := res.LastInsertId()
	vendorId := uint64(signedId)

	if stmt, err = tx1.Preparex("insert into auth (id, identifier_type, platform_type, identifier, seed, secret, ticks, private_code, session_id, session_footprint, delete_flag, create_at, update_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, utc_timestamp(), 0, utc_timestamp(), utc_timestamp())"); err != nil {
		return db.RollbackResolve(err, tx1)
	}
	if res, err = stmt.Exec(vendorId, reqBodyCreate.IdentifierType, platformType, reqBodyCreate.Identifier, reqBodyCreate.Seed, "", reqBodyCreate.Ticks, privateCode, sessionId); err != nil {
		return db.RollbackResolve(err, tx1)
	}
	if err = tx1.Commit(); err != nil {
		return db.RollbackResolve(err, tx1)
	}

	shard, err := db.Conns.Shard(vendorId)
	if err != nil { return err }
	var tx2 *sqlx.Tx
	if tx2, err = shard.Beginx(); err != nil { return err }
	stmt, err = tx2.Preparex(db.CreateVendorQuery(vendorId))
	if err != nil { return err }
	_, err = stmt.Exec()
	if err != nil { return err }
	stmt, err = tx2.Preparex(db.CreateQueueQuery(vendorId))
	if err != nil { return err }
	_, err = stmt.Exec()
	if err != nil { return err }
	stmt, err = tx2.Preparex(db.CreateKeycodeQuery(vendorId))
	if err != nil { return err }
	_, err = stmt.Exec()
	if err != nil { return err }
	stmt, err = tx2.Preparex(db.CreateAuthQuery(vendorId))
	if err != nil { return err }
	_, err = stmt.Exec()
	if err != nil { return err }
	// update set vendor info 
	if err != nil { return err }
	stmt, err = tx2.Preparex(`insert into vendor_` + db.ToSuffix(vendorId) + ` (
		id,
		queue_id,
		reset_count,
		name,
		first_code,
		last_code,
		total_wait,
		total_in,
		total_out,
		maintenance,
		caption, 
		delete_flag,
		create_at, 
		update_at
	) values (
		?,
		'',
		0, 
		?, 
		'',
		'',
		0,
		0,
		0,
		0,
		?,
		0,
		utc_timestamp(),
		utc_timestamp()
	);`)
	if err != nil { return err }
	_, err = stmt.Exec(vendorId, reqBodyCreate.Name, reqBodyCreate.Caption)
	if err != nil { return err }
	// commit or rollback
	if err := tx2.Commit(); err != nil {
		return db.RollbackResolve(err, tx2)
	}
	// update shard = -1 -> shard = proper_shard_num
	assigned_shard := db.GetShardNum(vendorId)
	stmt, err = master.Preparex("update domain set shard = ?, update_at = utc_timestamp() where id = ?")
	if err != nil { return err }
	_, err = stmt.Exec(assigned_shard, vendorId)
	if err != nil { return err }
	c.Echo().Logger.Debug("created")
	return c.String(http.StatusOK, "return master key and session_id here." + defs.ToBase64(sessionId))
}

// Logon vendor user request body struct
type ReqBodyLogon struct { }
// Logon vendor user response body struct
type ResBodyLogon struct { }
// Logon vendor user
func Logon(c echo.Context) error {
	// TODO require SSO check is ok.
	// create session id and update respond session_id
	return c.String(http.StatusOK, "return session_id here.")
}

// Update vendor user request body struct
type ReqBodyUpdate struct { }
// Update vendor user response body struct
type ResBodyUpdate struct { }
// Update vendor user
func Update(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Detail vendor user request body struct
type ReqBodyDetail struct { }
// Detail vendor user response body struct
type ResBodyDetail struct { }
// Detail vendor user
func Detail(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Add OAuth vendor user request body struct
type ReqBodyAddAuth struct { }
// Add OAuth vendor user response body struct
type ResBodyAddAuth struct { }
// Add OAuth vendor user
func AddAuth(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "return master key and session_id here.")
}

// Update OAuth vendor user request body struct
type ReqBodyUpdateAuth struct { }
// Update OAuth vendor user response body struct
type ResBodyUpdateAuth struct { }
// Update OAuth vendor user
func UpdateAuth(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "return master key and session_id here.")
}

// Initialize queue vendor user request body struct
type ReqBodyInitializeQueue struct { }
// Initialize queue vendor user response body struct
type ResBodyInitializeQueue struct { }
// Initialize queue vendor user
func InitializeQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	// generate keycodes here
	return c.String(http.StatusOK, "vendor")
}

// Show queue vendor user request body struct
type ReqBodyShowQueue struct { }
// Show queue vendor user response body struct
type ResBodyShowQueue struct { }
// Show queue vendor user
func ShowQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Update queue vendor user request body struct
type ReqBodyUpdateQueue struct { }
// Update queue vendor user response body struct
type ResBodyUpdateQueue struct { }
// Update queue vendor user
func UpdateQueue(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Logoff vendor user request body struct
type ReqBodyLogoff struct { }
// Logoff vendor user response body struct
type ResBodyLogoff struct { }
// Logoff vendor user
func Logoff(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}

// Purge vendor user request body struct
type ReqBodyPurge struct { }
// Purge vendor user response body struct
type ResBodyPurge struct { }
// Purge(logical remove) vendor user
func Purge(c echo.Context) error {
	// TODO require SSO check is ok.
	return c.String(http.StatusOK, "vendor")
}
