#!/bin/sh -x
#  The MIT License
#  Copyright (c) 2020 FurtherSystem Co.,Ltd.
#
#  Permission is hereby granted, free of charge, to any person obtaining a copy
#  of this software and associated documentation files (the "Software"), to deal
#  in the Software without restriction, including without limitation the rights
#  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
#  copies of the Software, and to permit persons to whom the Software is
#  furnished to do so, subject to the following conditions:
#
#  The above copyright notice and this permission notice shall be included in
#  all copies or substantial portions of the Software.
#
#  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
#  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
#  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
#  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
#  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
#  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
#  THE SOFTWARE.

DRYRUN=
DBCLIENT=mysql
DBADDR=localhost
DBUSER=root
DBPASS=
DBPREFIX=${1:-default}
GOTESTPREFIX=gotest
CREATE_USER=vql_user
CREATE_PASS=
CREATE_OPUSER=vql_opuser
CREATE_OPPASS=
PRODUCTION=${2:-0}

NUM_START=0x00
NUM_END=0x1f

die(){
  echo "$*"
  exit 1
}

create_db(){
  query="create database if not exists ${1} default character set utf8;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}"
}

create_table_domain(){
  query="use ${1};create table if not exists domain (
    id	 	bigint unsigned not null auto_increment,
    service_code        tinyint unsigned not null,
    vendor_code         varbinary(256) not null,
    shard 	smallint not null,
    delete_flag tinyint unsigned not null,
    create_at 	datetime not null,
    update_at 	datetime not null,
    primary key (id)
  ) engine=innodb;
"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}"
}

create_table_vendor_auth(){
  query="use ${1};create table if not exists auth (
    id                  bigint unsigned not null,
    identifier_type     tinyint unsigned not null,
    platform_type       varchar(128) not null,
    identifier          varchar(128) not null,
    seed                varchar(128) not null,
    secret              varchar(128) not null,
    ticks               bigint unsigned not null,
    private_code        varbinary(256) not null,
    session_id          varbinary(256) not null,
    session_private     varbinary(256) not null,
    session_footprint   datetime not null,
    delete_flag         tinyint unsigned not null,
    create_at           datetime not null,
    update_at           datetime not null,
    unique (identifier, seed)
  ) engine=innodb;
"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}"
}

create_user(){
  query="create user ${1}@'%' identified by \"${2}\";"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}"
}

grant_normal_db(){
  query="grant create, create view, create routine, drop, delete, index, insert, lock tables, select, update on ${1}.* to ${2}@'%';"
  flush="flush privileges;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}
${flush}"
}

grant_all_db(){
  query="grant all privileges on ${1}.* to ${2}@'%';"
  flush="flush privileges;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}
${flush}"
}

# main logics.

create_user ${CREATE_USER} ${CREATE_PASS} || die "error create user ${CREATE_USER}"
create_user ${CREATE_OPUSER} ${CREATE_OPPASS} || die "error create user ${CREATE_OPUSER}"
create_db ${DBPREFIX}_master || die "error create db ${DBPREFIX}_master"
grant_normal_db ${DBPREFIX}_master ${CREATE_USER} || die "error grant normal db ${DBPREFIX}_master"
grant_all_db ${DBPREFIX}_master ${CREATE_OPUSER} || die "error grant all db ${DBPREFIX}_master"
if [ "x${PRODUCTION}" = "x0" ];then
  grant_normal_db ${GOTESTPREFIX}_master ${CREATE_USER} || die "error grant normal db ${GOTESTPREFIX}_master"
  grant_all_db ${GOTESTPREFIX}_master ${CREATE_OPUSER} || die "error grant all db ${GOTESTPREFIX}_master"
fi
create_table_domain ${DBPREFIX}_master || die "erro create table vendor ${DBPREFIX}_master"
create_table_vendor_auth ${DBPREFIX}_master || die "erro create table vendor ${DBPREFIX}_master"

for suffix in `seq -w ${NUM_START} ${NUM_END}`
do
  hex_suffix=`printf '%02x' ${suffix}`
  create_db ${DBPREFIX}_shard_${hex_suffix} ${CREATE_USER} || die "error create db ${DBPREFIX}_shard_${hex_suffix}"
  grant_normal_db ${DBPREFIX}_shard_${hex_suffix} ${CREATE_USER} || die "error grant normal db ${DBPREFIX}_shard_${hex_suffix}"
  grant_all_db ${DBPREFIX}_shard_${hex_suffix} ${CREATE_OPUSER} || die "error grant all db ${DBPREFIX}_shard_${hex_suffix}"
  if [ "x${PRODUCTION}" = "x0" ];then
    grant_normal_db ${GOTESTPREFIX}_shard_${hex_suffix} ${CREATE_USER} || die "error grant normal db ${GOTESTPREFIX}_shard_${hex_suffix}"
    grant_all_db ${GOTESTPREFIX}_shard_${hex_suffix} ${CREATE_OPUSER} || die "error grant all db ${GOTESTPREFIX}_shard_${hex_suffix}"
  fi
done

echo "setup ok"

exit 0
