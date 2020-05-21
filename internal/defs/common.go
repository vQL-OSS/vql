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

package defs

import (
	"fmt"
	"math/rand"
)

// Version
var Version string

const (
	// Database prefix
	DBPrefix = "default"
	// Database master name
	DBMaster = "master"
	// Database shard name
	DBShard = "shard"
	// Database shard dividing counts
	DBShardDivide = 32
)

// Create newguid - 16 bytes array
func NewGuid() ([16]byte, error) {
	uuid := [16]byte{}
	_, err := rand.Read(uuid[:])
	if err != nil {
		return uuid, err
	}
	return uuid, nil
}

// Get guid hexstring from 16 bytes array
func GuidFormatString(guid [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", guid[0:4], guid[4:6], guid[6:8], guid[8:10], guid[10:])
}

// Resolve database master name
func DBMasterName() string {
	return DBPrefix + "_" + DBMaster
}

// Resolve database shard name
func DBShardName(num uint64) string {
	return fmt.Sprintf("%s_%s_%02x", DBPrefix, DBShard, num%DBShardDivide)
}

// Create table vendor query string
func CreateVendorQuery(suffix uint64) string {
	query := `
create table vendor_` + fmt.Sprintf("%016x", suffix) + ` (
    id			bigint unsigned not null,
    master_uuid		varchar(128) not null,
    master_key		varchar(128) not null,
    session_id		varchar(128) not null,
    session_footprint	datetime not null,
    queue_id		varchar(128) not null,
    reset_count		smallint unsigned not null,
    name		varchar(1024) not null,
    first_code		varchar(3) not null,
    last_code		varchar(3) not null,
    total_wait		smallint unsigned not null,
    total_in		smallint unsigned not null,
    total_out		smallint unsigned not null,
    maintenance		boolean not null,
    caption		varchar(4096) not null,
    delete_flag		tinyint unsigned not null,
    create_at		datetime not null,
    update_at		datetime not null,
    primary key (id)
) engine=innodb;`
	return query
}

// Drop table vendor query string
func DropVendorQuery(suffix uint64) string {
	query := `
drop table vendor_` + fmt.Sprintf("%016x", suffix) + `;`
	return query
}

// Create table queue query string
func CreateQueueQuery(suffix uint64) string {
	query := `
create table queue_` + fmt.Sprintf("%016x", suffix) + ` (
    id			bigint unsigned not null,
    queue_id		varchar(128) not null,
    keycode_prefix	varchar(3) not null,
    keycode_suffix	varchar(128) not null,
    session_id		varchar(128) not null,
    session_footprint	datetime not null,
    prev_code		varchar(3) not null,
    next_code		varchar(3) not null,
    mail		boolean not null,
    mail_addr		varchar(1024) not null,
    mail_count		smallint unsigned not null,
    push		boolean not null,
    push_type		tinyint unsigned not null,
    push_count		smallint unsigned not null,
    caption		varchar(1024) not null,
    delete_flag		tinyint unsigned not null,
    create_at		datetime not null,
    update_at		datetime not null,
    primary key (id),
    unique (keycode_prefix),
    unique (keycode_suffix)
  ) engine=innodb;`
	return query
}

// Drop table queue query string
func DropQueueQuery(suffix uint64) string {
	query := `
drop table queue_` + fmt.Sprintf("%016x", suffix) + `;`
	return query
}

// Create table keycode query string
func CreateKeyCodeQuery(suffix uint64) string {
	query := `
create table keycode_` + fmt.Sprintf("%016x", suffix) + ` (
    id			bigint unsigned not null,
    keycode_prefix	varchar(3) not null,
    keycode_suffix	varchar(128) not null,
    delete_flag		tinyint unsigned not null,
    create_at		datetime not null,
    update_at		datetime not null,
    primary key (id),
    unique (keycode_prefix),
    unique (keycode_suffix)
  ) engine=innodb;`
	return query
}

// Drop table keycode query string
func DropKeyCodeQuery(suffix uint64) string {
	query := `
drop table keycode_` + fmt.Sprintf("%016x", suffix) + `;`
	return query
}

// Create table auth query string
func CreateAuthQuery(suffix uint64) string {
	query := `
create table auth_` + fmt.Sprintf("%016x", suffix) + ` (
    id			bigint unsigned not null,
    privider_type	tinyint unsigned not null,
    uuid		varchar(128) not null,
    secret		varchar(128) not null,
    delete_flag		tinyint unsigned not null,
    create_at		datetime not null,
    update_at		datetime not null,
    primary key (id),
    unique (uuid)
  ) engine=innodb;`
	return query
}

// Drop table auth query string
func DropAuthQuery(suffix uint64) string {
	query := `
drop table auth_` + fmt.Sprintf("%016x", suffix) + `;`
	return query
}
