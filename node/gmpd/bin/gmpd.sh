#!/bin/bash

BASEDIR=$(cd "$(dirname "$0")"; pwd)
gmpd=gmpd.x64
os=$(uname -i)
if [[ os = i386 ]];then
  gmpd=gmpd.x32
fi
$BASEDIR/$gmpd $1 &
