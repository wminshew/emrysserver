#!bin/bash

mkdir -p /root/.ssh
touch /root/.ssh/authorized_keys
echo ${PUBLIC_KEY} > /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys

# rsyslogd
# rc-service syslog start

/usr/sbin/sshd -D -f ${SSHD_CONFIG} -e 2>&1
# /usr/sbin/sshd -D -f ${SSHD_CONFIG}
