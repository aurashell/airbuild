FROM ubuntu:18.04

WORKDIR /usr/src/airbuild

COPY *.go ./
COPY go.sum ./
COPY go.mod ./

RUN apt update && apt dist-upgrade -y
RUN apt install -y build-essential make cmake autoconf automake libtool meson ninja-build wget git python2.7

RUN echo 'deb http://apt.llvm.org/bionic/ llvm-toolchain-bionic-8 main ' >>/etc/apt/sources.list
RUN echo 'deb-src http://apt.llvm.org/bionic/ llvm-toolchain-bionic-8 main ' >>/etc/apt/sources.list
RUN wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | apt-key add -
RUN apt update && apt dist-upgrade -y
RUN apt install -y clang-8 lldb-8 lld-8

RUN wget https://dl.google.com/go/go1.12.7.linux-amd64.tar.gz -O go.tar.gz
RUN mkdir -p /usr/src/airbuild/go
RUN tar --strip-components=1 -xvf go.tar.gz -C /usr/src/airbuild/go
RUN git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git /usr/src/airbuild/depot_tools/
RUN mkdir -p /usr/src/airbuild/depot_tools_o/
RUN ln -s /usr/bin/python2.8 /usr/src/airbuild/depot_tools_o/
ENV PATH=/usr/src/airbuild/go/bin/:/usr/src/airbuild/gopath/bin/:/usr/src/airbuild/depot_tools/:/usr/src/airbuild/depot_tools_o/:$PATH
ENV GOPATH=/usr/src/airbuild/gopath/
ENV DEPOT_TOOLS_UPDATE=0
ENV CC=clang-8
ENV CXX=clang++-8
RUN go get && go install

WORKDIR /usr/src/app/

CMD ["/bin/bash"]
