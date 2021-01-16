FROM gcr.io/k8s-testimages/bootstrap:v20210108-5927ee692c

# set used shell
SHELL [ "/bin/bash", "-c"]
# install tiup
RUN curl --proto '=https' --tlsv1.2 -sSf https://tiup-mirrors.pingcap.com/install.sh | sh
RUN service docker start
ADD chingwei /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/chingwei" ]
