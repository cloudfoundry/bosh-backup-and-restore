FROM ubuntu:latest

RUN apt-get update
RUN apt-get install -y ssh
RUN apt-get install -y sudo
RUN mkdir /var/run/sshd
RUN echo 'root:p' | chpasswd
RUN sed -i 's/PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config

# SSH login fix. Otherwise user is kicked off after login
RUN sed 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' -i /etc/pam.d/sshd

ENV NOTVISIBLE "in users profile"
RUN echo "export VISIBLE=now" >> /etc/profile

COPY create_user_with_key /bin/create_user_with_key
RUN chmod +x /bin/create_user_with_key

# No pass for sudo
RUN echo "%sudo         ALL = (ALL) NOPASSWD: ALL" >> /etc/sudoers 

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]

