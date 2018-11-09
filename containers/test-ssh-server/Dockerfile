FROM gotechnies/alpine-ssh
COPY /files /
RUN chmod -R go-rwx /root/.ssh
CMD ["/usr/sbin/sshd","-D", "-e"]

