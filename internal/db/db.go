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

// Database server cluster access package
package db

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"time"
	"vql/internal/defs"
)

const (
	// Database name
	Name = "mysql"
	// Database master name
	Master = "master"
	// Database shard name
	Shard = "shard"
	// Database shard dividing counts
	ShardDivide = 32
	// Database normal user
	DefaultUser = "vql_user"
	// Database normal pass
	DefaultPass = "password"
	// Database operation user
	OperateUser = "vql_opuser"
	// Database operation pass
	OperatePass = "password"
	// Database master addr
	MasterAddr = "localhost"
	// Database master port
	MasterPort = "3306"
	// Database shard addr
	ShardAddr = "localhost"
	// Database shard port
	ShardPort = "3306"
	// max open connections
	MaxOpenConns = 20
	// max idle connections
	MaxIdleConns = 10
	// max life time connections
	ConnMaxLifetime = time.Hour
)

type Conn struct {
	master *sqlx.DB
	shard  [ShardDivide]*sqlx.DB
	user   string
	pass   string
}

var Conns = Conn{user: DefaultUser, pass: DefaultPass}
var OpConns = Conn{user: OperateUser, pass: OperatePass}

// Setup
func Setup() error {
	master, err := sqlx.Open(Name, OperateUser+":"+OperatePass+"@tcp("+MasterAddr+":"+MasterPort+")/")
	if err != nil {
		return err
	}
	defer master.Close()
	_, err = master.Exec(CreateDatabaseMaster())
	if err != nil {
		return err
	}
	_, err = master.Exec("use " + defs.ServicePrefix + "_" + Master)
	if err != nil {
		return err
	}
	//_, err = master.Exec(GrantNormal(defs.ServicePrefix + "_" + Master))
	//if err != nil {
	//	return err
	//}
	//_, err = master.Exec("flush privileges;")
	//if err != nil {
	//	return err
	//}
	tx, err := master.Beginx()
	if err != nil {
		return err
	}
	stmt, err := tx.Preparex(CreateDomainQuery())
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec()
	stmt, err = tx.Preparex(CreateVendorAuthQuery())
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	err = tx.Commit()

	for i := 0; i < ShardDivide; i++ {
		shard, err := sqlx.Open(Name, OperateUser+":"+OperatePass+"@tcp("+ShardAddr+":"+ShardPort+")/")
		if err != nil {
			return err
		}
		defer shard.Close()
		_, err = shard.Exec(CreateDatabaseShard(i))
		if err != nil {
			return err
		}
		_, err = shard.Exec("use " + fmt.Sprintf("%s_%s_%02x", defs.ServicePrefix, Shard, i))
		//_, err = shard.Exec(GrantNormal(fmt.Sprintf("%s_%s_%02x", defs.ServicePrefix, Shard, i)))
		//if err != nil {
		//	return err
		//}
		//_, err = shard.Exec("flush privileges;")
		//if err != nil {
		//	return err
		//}
	}
	return nil
}

// Teardown
func Teardown() error {
	master, err := sqlx.Open(Name, OperateUser+":"+OperatePass+"@tcp("+MasterAddr+":"+MasterPort+")/")
	if err != nil {
		return err
	}
	_, _ = master.Exec(DropDatabaseMaster())
	defer master.Close()

	for i := 0; i < ShardDivide; i++ {
		shard, err := sqlx.Open(Name, OperateUser+":"+OperatePass+"@tcp("+ShardAddr+":"+ShardPort+")/")
		if err != nil {
			return err
		}
		defer shard.Close()
		_, _ = shard.Exec(DropDatabaseShard(i))
	}
	return nil
}

// initialize all normal db connections
func (d *Conn) Init() error {
	var err error
	d.master, err = sqlx.Open(Name, d.user+":"+d.pass+"@tcp("+MasterAddr+":"+MasterPort+")/"+defs.ServicePrefix+"_"+Master)
	if err != nil {
		return err
	}
	d.master.SetMaxOpenConns(MaxOpenConns)
	d.master.SetMaxIdleConns(MaxIdleConns)
	d.master.SetConnMaxLifetime(ConnMaxLifetime)

	for i, _ := range d.shard {
		d.shard[i], err = sqlx.Open(Name, d.user+":"+d.pass+"@tcp("+ShardAddr+":"+ShardPort+")/"+fmt.Sprintf("%s_%s_%02x", defs.ServicePrefix, Shard, i))
		if err != nil {
			return err
		}
		d.shard[i].SetMaxOpenConns(MaxOpenConns)
		d.shard[i].SetMaxIdleConns(MaxIdleConns)
		d.shard[i].SetConnMaxLifetime(ConnMaxLifetime)
	}

	return nil
}

// Resolve database master
func (d *Conn) Master() *sqlx.DB {
	return d.master
}

// Resolve database shard
func (d *Conn) Shard(num uint64) (*sqlx.DB, error) {
	if num > ShardDivide {
		return nil, fmt.Errorf("error: shard num is over %d > %d", num, ShardDivide)
	}
	return d.shard[GetShardNum(num)], nil
}

func GetShardNum(num uint64) int {
	return int(num % ShardDivide)
}

func ToSuffix(num uint64) string {
	return fmt.Sprintf("%016x", num)
}

func RollbackResolve(err error, tx *sqlx.Tx) error {
	if res := tx.Rollback(); res != nil {
		err = fmt.Errorf("%s: %w", res.Error(), err)
	}
	return err
}

// Create database master
func CreateDatabaseMaster() string {
	query := `create database if not exists ` + defs.ServicePrefix + `_` + Master + ` default character set utf8;`
	return query
}

// Drop database master
func DropDatabaseMaster() string {
	query := `drop database ` + defs.ServicePrefix + `_` + Master + `;`
	return query
}

// Create database shard
func CreateDatabaseShard(suffix int) string {
	query := `create database if not exists ` + fmt.Sprintf("%s_%s_%02x", defs.ServicePrefix, Shard, suffix) + ` default character set utf8;`
	return query
}

// Drop database shard
func DropDatabaseShard(suffix int) string {
	query := `drop database ` + defs.ServicePrefix + `_` + Shard + `_` + fmt.Sprintf("%02x", suffix) + `;`
	return query
}

func GrantNormal(name string) string {
	query := "grant create, create view, delete, index, insert, lock tables, select, update on " + name + ".* to " + DefaultUser + "@'%';"
	return query
}

func GrantAll(name string) string {
	query := "grant all privileges on " + name + ".* to " + OperateUser + "@'%';"
	return query
}

// Create table master domain query string
func CreateDomainQuery() string {
	query := `
create table if not exists domain (
    id          	bigint unsigned not null auto_increment,
    service_code	tinyint unsigned not null,
    vendor_code		varbinary(256) not null,
    shard       smallint not null,
    delete_flag tinyint unsigned not null,
    create_at   datetime not null,
    update_at   datetime not null,
    primary key (id)
) engine=innodb;`
	return query
}

// Drop table domain query string
func DropDomainQuery() string {
	query := `drop table domain;`
	return query
}

// Domain table adaptor struct
type Domain struct {
	Id          uint64
	ServiceCode uint8  `db:"service_code"`
	VendorCode  []byte `db:"vendor_code"`
	Shard       int16
	DeleteFlag  uint8     `db:"delete_flag"`
	CreateAt    time.Time `db:"create_at"`
	UpdateAt    time.Time `db:"update_at"`
}

// Create table auth query string
func CreateVendorAuthQuery() string {
	query := `
create table auth (
    id			bigint unsigned not null,
    identifier_type	tinyint unsigned not null,
    platform_type	varchar(128) not null,
    identifier		varchar(128) not null,
    seed                varchar(128) not null,
    secret		varchar(128) not null,
    ticks		bigint unsigned not null,
    private_code	varbinary(256) not null,
    session_id          varbinary(256) not null,
    session_footprint   datetime not null,
    delete_flag		tinyint unsigned not null,
    create_at		datetime not null,
    update_at		datetime not null,
    unique (identifier, seed)
  ) engine=innodb;`
	return query
}

// Drop table auth query string
func DropVendorAuthQuery() string {
	query := `
drop table auth;`
	return query
}

// struct is same to consumer auth table.

// Create table summary query string
func CreateSummaryQuery(num uint64) string {
	query := `
create table summary_` + ToSuffix(num) + ` (
    id			bigint unsigned not null,
    queue_code  	varbinary(256) not null,
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
    primary key (id),
    unique (queue_code)
) engine=innodb;`
	return query
}

// Drop table vendor query string
func DropSummaryQuery(num uint64) string {
	query := `
drop table summary_` + ToSuffix(num) + `;`
	return query
}

// Vendor table adaptor struct
type Summary struct {
	Id          uint64
	QueueCode   []byte `db:"queue_code"`
	ResetCount  uint16 `db:"reset_count"`
	Name        string
	FirstCode   string `db:"first_code"`
	LastCode    string `db:"last_code"`
	TotalWait   uint16 `db:"total_wait"`
	TotalIn     uint16 `db:"total_in"`
	TotalOut    uint16 `db:"total_out"`
	Maintenance bool
	Caption     string
	DeleteFlag  uint8     `db:"delete_flag"`
	CreateAt    time.Time `db:"create_at"`
	UpdateAt    time.Time `db:"update_at"`
}

// Create table queue query string
func CreateQueueQuery(num uint64) string {
	query := `
create table queue_` + ToSuffix(num) + ` (
    id          	bigint unsigned not null auto_increment,
    queue_code  	varbinary(256) not null,
    uid			bigint unsigned not null,
    keycode_prefix	varchar(3) not null,
    keycode_suffix	varchar(128) not null,
    mail_addr		varchar(1024) not null,
    mail_count		smallint unsigned not null,
    push_type		tinyint unsigned not null,
    push_count		smallint unsigned not null,
    status		tinyint unsigned not null,
    delete_flag		tinyint unsigned not null,
    create_at		datetime not null,
    update_at		datetime not null,
    primary key (id),
    unique (queue_code, keycode_prefix)
  ) engine=innodb;`
	return query
}

// Drop table queue query string
func DropQueueQuery(num uint64) string {
	query := `
drop table queue_` + ToSuffix(num) + `;`
	return query
}

// Queue table adaptor struct
type Queue struct {
	Id            uint64
	QueueCode     []byte `db:"queue_code"`
	Uid           string `db:"uid"`
	KeycodePrefix string `db:"keycode_prefix"`
	KeycodeSuffix string `db:"keycode_suffix"`
	MailAddr      string `db:"mail_addr"`
	MailCount     uint16 `db:"mail_count"`
	PushType      uint8  `db:"push_type"`
	PushCount     uint16 `db:"push_count"`
	Status        uint8
	DeleteFlag    uint8     `db:"delete_flag"`
	CreateAt      time.Time `db:"create_at"`
	UpdateAt      time.Time `db:"update_at"`
}

// Create table keycode query string
func CreateKeycodeQuery(num uint64) string {
	query := `
create table keycode_` + ToSuffix(num) + ` (
    id			bigint unsigned not null,
    keycode_prefix	varchar(3) not null,
    keycode_suffix	varchar(128) not null,
    delete_flag		tinyint unsigned not null,
    create_at		datetime not null,
    update_at		datetime not null,
    primary key (id),
    unique (keycode_prefix, keycode_suffix)
  ) engine=innodb;`
	return query
}

// Drop table keycode query string
func DropKeycodeQuery(num uint64) string {
	query := `
drop table keycode_` + ToSuffix(num) + `;`
	return query
}

// Keycode table adaptor struct
type Keycode struct {
	Id            uint64
	KeycodePrefix string    `db:"keycode_prefix"`
	KeycodeSuffix string    `db:"keycode_suffix"`
	DeleteFlag    uint8     `db:"delete_flag"`
	CreateAt      time.Time `db:"create_at"`
	UpdateAt      time.Time `db:"update_at"`
}

// Create table auth query string
func CreateAuthQuery(num uint64) string {
	query := `
create table auth_` + ToSuffix(num) + ` (
    id			bigint unsigned not null,
    platform_type	varchar(128) not null,
    identifier_type	tinyint unsigned not null,
    identifier		varchar(128) not null,
    seed                varchar(128) not null,
    secret		varchar(128) not null,
    ticks		bigint unsigned not null,
    private_code	varbinary(256) not null,
    session_id          varbinary(256) not null,
    session_footprint   datetime not null,
    delete_flag		tinyint unsigned not null,
    create_at		datetime not null,
    update_at		datetime not null,
    primary key (id),
    unique (identifier, seed)
  ) engine=innodb;`
	return query
}

// Drop table auth query string
func DropAuthQuery(num uint64) string {
	query := `
drop table auth_` + ToSuffix(num) + `;`
	return query
}

// Auth table adaptor struct
type Auth struct {
	Id               uint64
	PlatformType     string `db:"platform_type"`
	IdentifierType   uint8  `db:"identifier_type"`
	Identifier       string
	Seed             string
	Secret           string
	PrivateCode      []byte    `db:"private_code"`
	SessionId        string    `db:"session_id"`
	SessionFootprint time.Time `db:"session_footprint"`
	DeleteFlag       uint8     `db:"delete_flag"`
	CreateAt         time.Time `db:"create_at"`
	UpdateAt         time.Time `db:"update_at"`
}

// Create table sequence string
func CreateSequenceQuery(num uint64) string {
	query := `
create table sequence_` + ToSuffix(num) + ` (
    name		char(3) not null,
    curr_value		bigint unsigned not null,
    increment		smallint not null,
    primary key (name)
  ) engine=innodb;`
	return query
}

// Drop table sequence query string
func DropSequenceQuery(num uint64) string {
	query := `
drop table sequence_` + ToSuffix(num) + `;`
	return query
}

// Sequence table adaptor struct
type Sequence struct {
	Name             string
	CurrValue        uint64 `db:"curr_value"`
	Increment        int16
}

func NewSequenceQuery(num uint64) string {
	query := `
insert into sequence_` + ToSuffix(num) + ` values (?, ?, ?);`
	return query
}

func CreateFuncCurrSeqQuery(num uint64) string {
	query := `
create function currseq_` + ToSuffix(num) + ` (seq_name char(3))
    returns bigint unsigned
    language sql
    deterministic
    contains sql
    sql security definer
    comment ''
begin
    declare value bigint unsigned;
    set value = 0;
    select curr_value into value from sequence_` + ToSuffix(num) + ` where name = seq_name;
    return value;
end`
	return query
}

func CreateFuncNextSeqQuery(num uint64) string {
	query := `
create function nextseq_` + ToSuffix(num) + ` (seq_name char(3))
    returns bigint unsigned
    language sql
    deterministic
    contains sql
    sql security definer
    comment ''
begin
    update sequence_` + ToSuffix(num) + ` set curr_value = curr_value + increment where name = seq_name;
    return currseq_` + ToSuffix(num) + `(seq_name);
end`
	return query
}

func CreateFuncUpdateSeqQuery(num uint64) string {
	query := `
create function setseq_` + ToSuffix(num) + ` (seq_name char(3), value integer)
    returns bigint unsigned
    language sql
    deterministic
    contains sql
    sql security definer
    comment ''
begin
    update sequence_` + ToSuffix(num) + ` set curr_value = value where name = seq_name;
    return currseq` + ToSuffix(num) + `(seq_name);
end`
	return query
}
