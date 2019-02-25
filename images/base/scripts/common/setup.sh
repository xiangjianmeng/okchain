#!/bin/bash

# ALERT: if you encounter an error like:
# error: [Errno 1] Operation not permitted: 'cf_update.egg-info/requires.txt'
# The proper fix is to remove any "root" owned directories under your update-cli directory
# as source mount-points only work for directories owned by the user running vagrant

# Stop on first error
set -e
set -x

# Update the entire system to the latest releases
apt-get update -qq
apt-get dist-upgrade -qqy

# install git
apt-get install --yes git

MACHINE=`uname -m`
if [ x$MACHINE = xppc64le ]
then
   # install sudo
   apt-get install --yes sudo
fi

apt-get -y install llvm g++ libgmp-dev libssl-dev

# Set Go environment variables needed by other scripts
export GOPATH="/opt/gopath"

#install golang
#apt-get install --yes golang
mkdir -p $GOPATH
if [ x$MACHINE = xs390x ]
then
   apt-get install --yes golang
   export GOROOT="/usr/lib/go-1.6"
elif [ x$MACHINE = xppc64le ]
then
   wget ftp://ftp.unicamp.br/pub/linuxpatch/toolchain/at/ubuntu/dists/trusty/at9.0/binary-ppc64el/advance-toolchain-at9.0-golang_9.0-3_ppc64el.deb
   dpkg -i advance-toolchain-at9.0-golang_9.0-3_ppc64el.deb
   rm advance-toolchain-at9.0-golang_9.0-3_ppc64el.deb

   update-alternatives --install /usr/bin/go go /usr/local/go/bin/go 9
   update-alternatives --install /usr/bin/gofmt gofmt /usr/local/go/bin/gofmt 9

   export GOROOT="/usr/local/go"
elif [ x$MACHINE = xx86_64 ]
then
   export GOROOT="/opt/go"

   #ARCH=`uname -m | sed 's|i686|386|' | sed 's|x86_64|amd64|'`
   ARCH=amd64
   GO_VER=1.11

   cd /tmp
#   wget --quiet --no-check-certificate https://storage.googleapis.com/golang/go$GO_VER.linux-${ARCH}.tar.gz
   wget --quiet --no-check-certificate https://studygolang.com/dl/golang/go$GO_VER.linux-${ARCH}.tar.gz
   tar -xvf go$GO_VER.linux-${ARCH}.tar.gz
   mv go $GOROOT
   chmod 775 $GOROOT
   rm go$GO_VER.linux-${ARCH}.tar.gz
else
  echo "TODO: Add $MACHINE support"
  exit
fi

PATH=$GOROOT/bin:$GOPATH/bin:$PATH

cat <<EOF >/etc/profile.d/goroot.sh
export GOROOT=$GOROOT
export GOPATH=$GOPATH
export PATH=\$PATH:$GOROOT/bin:$GOPATH/bin
export LD_LIBRARY_PATH=/lib:/usr/lib:/usr/local/lib
EOF

# Install BLS
TEMP_DIR=/tmp
cd $TEMP_DIR
git clone https://github.com/dfinity/bn
cd bn
make
make install


# Install GRPC
# ----------------------------------------------------------------
# NOTE: For instructions, see https://github.com/google/protobuf
#
# ----------------------------------------------------------------

# First install protoc
cd /tmp
wget --quiet https://github.com/google/protobuf/archive/v3.0.2.tar.gz
tar xpzf v3.0.2.tar.gz
cd protobuf-3.0.2
apt-get install -y autoconf automake libtool curl make g++ unzip
apt-get install -y build-essential
./autogen.sh
# NOTE: By default, the package will be installed to /usr/local. However, on many platforms, /usr/local/lib is not part of LD_LIBRARY_PATH.
# You can add it, but it may be easier to just install to /usr instead.
#
# To do this, invoke configure as follows:
#
# ./configure --prefix=/usr
#
#./configure
./configure --prefix=/usr

if [ x$MACHINE = xs390x ]
then
    echo FIXME: protobufs wont compile on 390, missing atomic call
else
    make
    make check
    make install
fi
export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH
cd ~/

# Install rocksdb
apt-get install -y libsnappy-dev zlib1g-dev libbz2-dev
cd /tmp
git clone https://github.com/facebook/rocksdb.git
cd rocksdb
git checkout tags/v5.10.4
if [ x$MACHINE = xs390x ]
then
    echo There were some bugs in 4.1 for z/p, dev stream has the fix, living dangereously, fixing in place
    sed -i -e "s/-march=native/-march=z196/" build_tools/build_detect_platform
    sed -i -e "s/-momit-leaf-frame-pointer/-DDUMBDUMMY/" Makefile
elif [ x$MACHINE = xppc64le ]
then
    echo There were some bugs in 4.1 for z/p, dev stream has the fix, living dangereously, fixing in place.
    echo Below changes are not required for newer releases of rocksdb.
    sed -ibak 's/ifneq ($(MACHINE),ppc64)/ifeq (,$(findstring ppc64,$(MACHINE)))/g' Makefile
fi

PORTABLE=1 make shared_lib
INSTALL_PATH=/usr/local make install-shared
ldconfig
cd ~/

# Make our versioning persistent
echo $BASEIMAGE_RELEASE > /etc/okchain-baseimage-release

# clean up our environment
apt-get -y autoremove
apt-get clean
rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
