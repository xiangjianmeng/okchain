## OKChain
The official Golang implementation of the OKChain protocol.

Overview - [Homepage](https://www.okcoin.com/chain)
==========================

OKChain is a high-performance public distributed multi-chain platform and a private, secure, decentralised digital currency. 

**Privacy:** OKChain uses a cryptographically sound system to allow you to send and receive funds without your transactions being easily revealed on the blockchain (the ledger of transactions that everyone has). This ensures that your purchases, receipts, and all transfers remain absolutely private by default.

**Security:** Using the power of a distributed peer-to-peer consensus network, every transaction on the network is cryptographically secured. Individual wallets have a 25 word mnemonic seed that is only displayed once, and can be written down to backup the wallet. Wallet files are encrypted with a passphrase to ensure they are useless if stolen.

This is the core implementation of OKChain. It is open source and completely free to use without restrictions, except for those specified in the license agreement below. There are no restrictions on anyone creating an alternative implementation of OKChain that uses the protocol and network in a compatible manner.

As with many development projects, the repository on Github is considered to be the "staging" area for the latest changes. Before changes are merged into that branch on the main repository, they are tested by individual developers in their own branches, submitted as a pull request, and then subsequently tested by contributors who focus on testing and code reviews. That having been said, the repository should be carefully considered before using it in a production environment, unless there is a patch in the repository for a particular show-stopping issue you are experiencing. It is generally a better idea to use a tagged release for stability.

## Features
* Sharding, including network and transaction sharding
* PoW or VRSF for joining the network
* BLS-pbft consensus mechanism
* Coinbase rewards
* Ecc-p256 signature and public address
* Supporting ethereum smart contracts
* Gossip protocol
* Node recovery mechanism


## Installation

### Supported Operating System

| Operating System      | Processor | Status |
| --------------------- | --------  |--------|
| Ubuntu 16.04          |  amd64    | Supported
| OSX 10.13             |  amd64    | Supported
| Centos 7              |  amd64    | Supported
| Windows 10            |  amd64    | 2019 Q2

### Dependencies
The following table summarizes the tools and libraries required to build. A
few of the libraries are also included in this repository (marked as
"Vendored"). By default, the build uses the library installed on the system,
and ignores the vendored sources. However, if no library is found installed on
the system, then the vendored source will be built and used. The vendored
sources are also used for statically-linked builds because distribution
packages often include only shared library binaries (`.so`) but not static
library archives (`.a`).

| Dep          | Min. version  | Vendored | Debian/Ubuntu pkg  | Arch pkg     | Fedora  | Optional | Purpose  |
| ------------ | ------------- | -------- | ------------------ | ------------ | ------- | -------- | -------- |
| RocksDB      | 5.10.4        | NO       | `build-essential`  | `base-devel` | `gcc`   | NO       |          |               | ------------ | ------------- | -------- | ------------------ | ------------ | ------- | -------- | -------- |
| BLS          | NO        	 | NO       | `llvm libgmp-dev libssl-dev` | NO | `g++`   | NO       |          |
#### Build RocksDB

Upgrade your gcc to version at least 4.8 to get C++11 support.
##### On Ubuntu

* Install Dependencies

        apt-get update
        apt-get -y install build-essential libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev gcc
* Install rocksdb

        wget https://github.com/facebook/rocksdb/archive/v5.10.4.zip
        unzip rocksdb-5.10.4.zip
        cd rocksdb-5.10.4
        make shared_lib && make install-shared
##### On CentOS

* Install Dependencies

        yum -y install epel-release && yum -y update
        yum -y install gflags-devel snappy-devel zlib-devel bzip2-devel gcc-c++  libstdc++-devel
* Install rocksdb

        wget https://github.com/facebook/rocksdb/archive/v5.10.4.zip
        unzip rocksdb-5.10.4.zip
        cd rocksdb-5.10.4
        make shared_lib && make install-shared

##### On Mac OSX

* Install Dependencies

        brew install gcc gflags lz4 snappy
* Install rocksdb

        wget https://github.com/facebook/rocksdb/archive/v5.10.4.zip
        unzip rocksdb-5.10.4.zip
        cd rocksdb-5.10.4
        make shared_lib && make install-shared

    

#### Build BLS
##### On Ubuntu

* Install Dependencies

        apt-get update
        apt-get -y install llvm g++ libgmp-dev libssl-dev 
        echo 'export PATH="/usr/local/opt/llvm/bin:$PATH"' >> ~/.bash_profile
        source ~/.bash_profile
* Install BLS
        
	To install from source, you can run the following commands:

        git clone https://github.com/dfinity/bn
        cd bn
        make && make install
        export LD_LIBRARY_PATH=/lib:/usr/lib:/usr/local/lib
        
The library is then installed under `/usr/local/`.
 - To use it for the current session:
```bash
export LD_LIBRARY_PATH=/lib:/usr/lib:/usr/local/lib
```
 - To use it permanently, add `/usr/local/lib` to `/etc/ld.so.conf` , then run `ldconfig` as root

##### On Mac OSX

* Install Dependencies

        brew install llvm gmp g++ openssl 
        echo 'export PATH="/usr/local/opt/llvm/bin:$PATH"' >> ~/.bash_profile
        source ~/.bash_profile
* Install BLS
        
	To install from source, you can run the following commands:

        git clone https://github.com/dfinity/bn
        cd bn
        make && make install
        export LD_LIBRARY_PATH=/lib:/usr/lib:/usr/local/lib
        

### Build OKChain from Source code
First, prepare the Go compiler:

```
brew install go
```

Then, clone the `OKChain` repository to a specified directory:

```
git clone https://github.com/ok-chain/okchain
```

Finally, build the `OKChaind` and `OKChaincli` program using the following command.

```
cd okchain
mkdir -p build/bin

cd cmd/okchaind
GOBIN=${GOPATH}/src/github.com/ok-chain/okchain/build/bin go install

cd cmd/okchaincli
GOBIN=${GOPATH}/src/github.com/ok-chain/okchain/build/bin go install
```

Note: `okchaind` is the OKChain daemon program. And `OKChaincli` is a command-line tool to interacte with the `OKChaind`.

## Run OKChain
### Run okchaind using binaries

The build places the binary in `bin/` sub-directory within the build directory
from which make was invoked (repository root by default). To run in
foreground:

    ./okchaind


To run in background:

    ./okchaind --detach

### Run okchaind from docker

* Docker quick start

        docker run -d --name okchain/testnet-okchaind -v /okchain/chain:/root/.okchain \
                -p 15000:15000 -p 16000:16000  -p 25000:25000 \
                okchaind

* Startup a local testnet 

        cd dev/docker
        docker-compose up -d

## More information

- Web: [okcoin.com/chain](https://www.okcoin.com/chain)
- Forum: [forum.okchain.org](https://forum.okchain.org)
- Mail: [dev@okcoin.com](mailto:dev@okcoin.com)
- GitHub: [https://github.com/ok-chain/okchain](https://github.com/ok-chain/okchain)


## Contributing

**Anyone is welcome to contribute to OKChain's codebase!** If you have a fix or code change, feel free to submit it as a pull request directly to the "master" branch. In cases where the change is relatively small or does not affect other parts of the codebase it may be merged in immediately by any one of the collaborators. On the other hand, if the change is particularly large or complex, it is expected that it will be discussed at length either well in advance of the pull request being submitted, or even directly on the pull request.

If you want to help out, see [CONTRIBUTING](CONTRIBUTING.md) for a set of guidelines.

Packaging for your favorite distribution would be a welcome contribution!

## License

Copyright (c) 2018-2019 The OKChain Project.   
More details, See [LICENSE](LICENSE).

