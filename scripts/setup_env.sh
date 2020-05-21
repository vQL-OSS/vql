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

source `dirname $0`/common.env

source ~/.bash_profile
HOME_PATH=${HOME}

yum install -y gcc
yum install -y rpm-build

curl https://dl.google.com/go/go1.13.8.linux-amd64.tar.gz | tar zx -C ~/
echo export GO111MDULE=on >> ~/.bash_profile
echo export GOPATH=~/go >> ~/.bash_profile
echo 'export PATH=${PATH}:${GOPATH}/bin' >> ~/.bash_profile
echo export PATH >> ~/.bash_profile
source ~/.bash_profile
