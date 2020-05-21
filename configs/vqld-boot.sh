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

