#!bin/bash
# inspiration: http://blog.bonesoul.com/jumpserver-kubernetes/

ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa
ssh-keygen -f /etc/ssh/ssh_host_ecdsa_key -N '' -t ecdsa
ssh-keygen -f /etc/ssh/ssh_host_ed25519_key -N '' -t ed25519

mkdir -p /root/.ssh
touch /root/.ssh/authorized_keys
echo ${PUBLIC_KEY} > /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys

/usr/sbin/sshd -D -f ${SSHD_CONFIG}
# /usr/sbin/sshd -D -p 2222
