FROM docker.io/library/ubuntu:22.04

RUN sed -i 's/archive.ubuntu.com/mirrors.aliyun.com/g' /etc/apt/sources.list
RUN sed -i 's/security.ubuntu.com/mirrors.aliyun.com/g' /etc/apt/sources.list


RUN apt-get clean && apt-get update && apt-get install -y \
        build-essential \
        wget \
        tar

ENV PATH="/usr/local/bin:$PATH"
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        wget -O code-server.tar.gz https://github.com/coder/code-server/releases/download/v4.95.3/code-server-4.95.3-linux-amd64.tar.gz; \
    elif [ "$ARCH" = "aarch64" ]; then \
        wget -O code-server.tar.gz https://github.com/coder/code-server/releases/download/v4.95.3/code-server-4.95.3-linux-arm64.tar.gz; \
    else \
        echo "Unsupported architecture: $ARCH"; \
        exit 1; \
    fi && \
    mkdir -p /usr/local/code-server && \
    tar -xzvf code-server.tar.gz --strip-components=1 -C /usr/local/code-server && \
    rm code-server.tar.gz

# build and install skalibs、execline and s6
ARG SKALIBS_VERSION=2.14.3.0
ARG EXECLINE_VERSION=2.9.6.1
ARG S6_VERSION=2.13.1.0

RUN wget https://skarnet.org/software/skalibs/skalibs-${SKALIBS_VERSION}.tar.gz && \
    tar -xzf skalibs-${SKALIBS_VERSION}.tar.gz && \
    cd skalibs-${SKALIBS_VERSION} && \
    ./configure --prefix=/usr/local --enable-static --disable-shared --enable-static-libc && \
    make && make install && cd .. && rm -rf skalibs-${SKALIBS_VERSION}*

RUN wget https://skarnet.org/software/execline/execline-${EXECLINE_VERSION}.tar.gz && \
    tar -xzf execline-${EXECLINE_VERSION}.tar.gz && \
    cd execline-${EXECLINE_VERSION} && \
    ./configure --prefix=/usr/local --enable-static --disable-shared --enable-static-libc && \
    make && make install && cd .. && rm -rf execline-${EXECLINE_VERSION}*

RUN wget https://skarnet.org/software/s6/s6-${S6_VERSION}.tar.gz && \
    tar -xzf s6-${S6_VERSION}.tar.gz && \
    cd s6-${S6_VERSION} && \
    ./configure --prefix=/usr/local --enable-static --disable-shared --enable-static-libc && \
    make LDFLAGS="--static" && make install && cd .. && rm -rf s6-${S6_VERSION}*

RUN apt-get clean && rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["ls"]