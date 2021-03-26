FROM debian:buster-slim

RUN apt-get -qq update
RUN DEBIAN_FRONTEND=noninteractive apt-get -qq install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests -y bash openssh-server vim

RUN mkdir /run/sshd && chmod 755 /run/sshd
RUN /usr/bin/ssh-keygen -A
RUN ssh-keygen -t rsa -b 4096 -f  /etc/ssh/ssh_host_key

EXPOSE 22
COPY /files /
RUN chmod -R go-rwx /root/.ssh
CMD ["/usr/sbin/sshd","-D", "-e"]

