FROM telegraf

COPY getimages.sh /tmp/getimages.sh
COPY main.py /tmp/main.py
#COPY myjson.json /tmp/myjson.json
RUN echo 'Acquire::ForceIPv4 "true";' | tee /etc/apt/apt.conf.d/99force-ipv4 && \
apt-get update && \
apt-get -y install curl jq python && \
chmod +x /tmp/getimages.sh && \
mkdir /docker
