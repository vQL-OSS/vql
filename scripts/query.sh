#!/bin/sh
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
DBPREFIX=default
CREATE_USER=vql_user
CREATE_PASS=
CREATE_OPUSER=vql_opuser
CREATE_OPPASS=

NUM_START=0x00
NUM_END=0x1f

die(){
  echo "$*"
  exit 1
}

select_domain(){
	query="select id, service_code, to_base64(vendor_code), shard, delete_flag, create_at, update_at from ${DBPREFIX}_master.domain where id between ${1} and ${1}+${2} order by id;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}" -s
}

select_vauth(){
	query="select id, identifier_type, platform_type, identifier, seed, secret, ticks, to_base64(private_code), to_base64(session_id), session_footprint, delete_flag, create_at, update_at from ${DBPREFIX}_master.auth where id between ${1} and ${1}+${2} order by id;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}" -s
}

select_auth(){
  dbsuffix=`printf '%02x' ${1}`
  tablesuffix=`printf '%016x' ${1}`
  query="select * from ${DBPREFIX}_shard_${dbsuffix}.auth_${tablesuffix} where id between ${2} and ${2}+${3} order by id;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}" -s
}

select_keycode(){
  dbsuffix=`printf '%02x' ${1}`
  tablesuffix=`printf '%016x' ${1}`
  query="select * from ${DBPREFIX}_shard_${dbsuffix}.keycode_${tablesuffix} where id between ${2} and ${2}+${3} order by id;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}" -s
}

select_queue(){
  dbsuffix=`printf '%02x' ${1}`
  tablesuffix=`printf '%016x' ${1}`
  query="select * from ${DBPREFIX}_shard_${dbsuffix}.queue_${tablesuffix} where id between ${2} and ${2}+${3} order by id;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}" -s
}

select_vendor(){
  dbsuffix=`printf '%02x' ${1}`
  tablesuffix=`printf '%016x' ${1}`
  query="select * from ${DBPREFIX}_shard_${dbsuffix}.vendor_${tablesuffix} where id between ${2} and ${2}+${3} order by id;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}" -s
}

ident(){
  query="select * from ${DBPREFIX}_master.vendor where id between ${1} and ${1}+${2} order by id;"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}" -s
}


usage(){
  echo "[USAGE] domain|auth|keycode|queue|vendor start length"
  echo "[USAGE] ident uuid"
  echo "[USAGE] domain_ident|auth_ident|keycode_ident|queue_ident|vendor_ident"
}

# main logics.

case "$1" in
  domain ) select_domain $2 $3 ;;
  vauth ) select_vauth $2 $3 ;;
  auth ) select_auth $2 $3 $4 ;;
  keycode ) select_keycode $2 $3 $4 ;;
  queue ) select_queue $2 $3 $4 ;;
  vendor ) select_vendor $2 $3 $4 ;;
  ident ) select_ident $2 ;;
  * ) usage ;;
esac

exit 0
