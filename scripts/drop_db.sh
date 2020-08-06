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
DBGOTEST=gotest
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

drop_db(){
  query="drop database ${1};"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}"
}

drop_user(){
  query="drop user ${1}@'%';"
  ${DRYRUN} ${DBCLIENT} -u${DBUSER} -h${DBADDR} -p${DBPASS} -e "${query}"
}

# main logics.
drop_user ${CREATE_USER} || die "error drop user ${DROP_USER}"
drop_user ${CREATE_OPUSER} || die "error drop user ${DROP_OPUSER}"
drop_db ${DBPREFIX}_master || die "error drop db ${DBPREFIX}_master"
drop_db ${DBGOTEST}_master 2>/dev/null

for suffix in `seq -w ${NUM_START} ${NUM_END}`
do
  hex_suffix=`printf '%02x' ${suffix}`
  drop_db ${DBPREFIX}_shard_${hex_suffix} || die "error drop db ${DBPREFIX}_shard_${hex_suffix}"
  drop_db ${DBGOTEST}_shard_${hex_suffix} 2>/dev/null
done

echo "remove ok"

exit 0
