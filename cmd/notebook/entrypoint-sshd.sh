#!bin/bash

mkdir -p /.ssh
touch /.ssh/authorized_keys
chmod 0755 /.ssh
chmod 0604 /.ssh/authorized_keys
echo ${PUBLIC_KEY} > /.ssh/authorized_keys

/usr/sbin/sshd -D -f ${SSHD_CONFIG} -e 2>&1
