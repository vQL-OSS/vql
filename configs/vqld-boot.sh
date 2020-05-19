#!/bin/sh
# Copyright (c) 2018 FurtherSystem Co.,Ltd. All rights reserved.
#
#   This program is free software; you can redistribute it and/or modify
#   it under the terms of the GNU General Public License as published by
#   the Free Software Foundation; version 2 of the License.
#
#   This program is distributed in the hope that it will be useful,
#   but WITHOUT ANY WARRANTY; without even the implied warranty of
#   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#   GNU General Public License for more details.
#
#   You should have received a copy of the GNU General Public License
#   along with this program; if not, write to the Free Software
#   Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1335  USA

ENV_FILE=/etc/sysconfig/vqld.env
source $ENV_FILE

for OPT in "$@"
do
    case $OPT in
        -dryrun)
            DRYRUN=$2
            shift 2
            ;;
        -config_test)
            CONFIG_TEST=$2
            shift 2
            ;;
        -listen_addr)
            LISTEN_ADDR=$2
            shift 2
            ;;
        -listen_port)
            LISTEN_PORT=$2
            shift 2
            ;;
        -versiononly)
            VERSIONONLY=$2
            shift 2
            ;;
        -)
            shift 1
            break
            ;;
        -*)
            echo "$PROGNAME: illegal option -- '$(echo $1 | sed 's/^-*//')'" 1>&2
            exit 1
            ;;
        *)
            shift 1
            ;;
    esac
done

${DRYRUN} ${IMAGE_PATH}/${IMAGE_NAME} \
-config_test=${CONFIG_TEST} \
-listen_addr=${LISTEN_ADDR} \
-listen_port=${LISTEN_PORT} \
-versiononly=${VERSIONONLY}

