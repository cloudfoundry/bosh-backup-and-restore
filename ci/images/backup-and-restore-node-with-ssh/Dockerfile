FROM pcfplatformrecovery/backup-and-restore-bosh-stemcell:latest

RUN mkdir -p /var/vcap/store
RUN mkdir -p /var/vcap/jobs

RUN mkdir /var/run/sshd
RUN /usr/bin/ssh-keygen -A

COPY create_user_with_key /bin/create_user_with_key
RUN chmod +x /bin/create_user_with_key

# No pass for sudo
RUN echo "%sudo         ALL = (ALL) NOPASSWD: ALL" >> /etc/sudoers

EXPOSE 22
ENV PATH /var/vcap/bosh/bin:$PATH
CMD ["/usr/sbin/sshd", "-D"]
