# <h1 align="center">CESS-BUCKET &middot; [![GitHub license](https://img.shields.io/badge/license-Apache2-blue)](#LICENSE) <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.16-blue.svg" /></a></h1>

cess-bucket is a mining program provided by cess platform for storage miners.



## Reporting a Vulnerability

If you find out any vulnerability, Please send an email to tech@cess.one, we are happy to communicate with you.



## System Requirements

- Linux-amd64



## System dependencies

**<1> Install common libraries**

Take the ubuntu distribution as an example:

```
sudo apt upgrade -y && sudo apt install m4 g++ flex bison make gcc git curl wget lzip vim util-linux -y
```

**<2> Install the necessary pbc library**

```
sudo wget https://gmplib.org/download/gmp/gmp-6.2.1.tar.lz
sudo lzip -d gmp-6.2.1.tar.lz
sudo tar -xvf gmp-6.2.1.tar
cd gmp-6.2.1/
sudo chmod +x ./configure
sudo ./configure --enable-cxx
sudo make
sudo make check
sudo make install
cd ..

sudo wget https://crypto.stanford.edu/pbc/files/pbc-0.5.14.tar.gz
sudo tar -zxvf pbc-0.5.14.tar.gz
cd pbc-0.5.14/
sudo chmod +x ./configure
sudo ./configure
sudo make
sudo make install
sudo touch /etc/ld.so.conf.d/libpbc.conf
sudo test -s /etc/ld.so.conf.d/libpbc.conf
sudo sed -i "\$a /usr/local/lib" /etc/ld.so.conf.d/libpbc.conf || echo "/usr/local/lib" >> /etc/ld.so.conf.d/libpbc.conf
sudo ldconfig
```



## System configuration

- Firewall

If the firewall is turned on, you need to open the running port, the default port is 15001.

Take the ubuntu distribution as an example:

```
sudo ufw allow 15001/tcp
```
- Network optimization (optional)

```
sysctl -w net.ipv4.tcp_syncookies = 1
sysctl -w net.ipv4.tcp_tw_reuse = 1
sysctl -w net.ipv4.tcp_tw_recycle = 1
sysctl -w net.ipv4.tcp_fin_timeout = 30
sysctl -w net.ipv4.tcp_keepalive_time = 120
sysctl -w net.core.rmem_max=2500000
sysctl -w net.ipv4.ip_local_port_range = 1024 65500
```



## Build from source

**<1> Install go locale**

CESS-Bucket requires [Go 1.16.5](https://golang.org/dl/) or higher.

> See the [official Golang installation instructions](https://golang.org/doc/install) If you get stuck in the following process.

- Download go1.16.5 compress the package and extract it to the /use/local directory:

```bash
wget -c https://golang.org/dl/go1.16.5.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local
```

- You'll need to add `/usr/local/go/bin` to your path. For most Linux distributions you can run something like:

```shell
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc && source ~/.bashrc
```

- View your go version:

```
go version
```

**<2> Build a bucket**

```
git clone https://github.com/CESSProject/cess-bucket
cd cess-bucket/
go build -o bucket cmd/main/main.go
```

If all goes well, you will get a mining program called `bucket`.



# **Get started with bucket**

**<1> Register two polka wallet**

- For wallet one, it is called an  `income account`, which is used to receive rewards from mining, and you should keep the private key carefully.
- For wallet two, it is called a `signature account`, which is used to sign on-chain transactions. You need to recharge the account with a small tokens and provide the private key to the miner's configuration file. The cess system will not record and destroy the account.

Browser access: [App](https://testnet-rpc.cess.cloud/explorer) implemented by [CESS Explorer](https://github.com/CESSProject/cess-explorer), [Add two accounts](https://github.com/CESSProject/W3F-illustration/blob/main/gateway/createAccount.PNG) in two steps.

**<2> Recharge your signature account**

- If you are using the test network, Please join the [CESS discord](https://discord.gg/mYHTMfBwNS) to get it for free.
- If you are using the official network, please buy CESS tokens.

**<3> Prepare configuration file**

Use bucket to directly generate configuration file template:

```
chmod +x bucket
sudo ./bucket default
```

The content of the configuration file template is as follows. You need to fill in your own information into the file. By default, the `bucket` uses `conf.toml` in the current directory as the runtime configuration file. You can use `-c` or `--config` to specify the configuration file Location.

> Our testnet rpc address is: `wss://testnet-rpc.cess.cloud/ws/`

```
# The rpc address of the chain node
RpcAddr      = ""
# Path to the mounted disk where the data is saved
MountedPath  = ""
# Total space used to store files, the unit is GB
StorageSpace = 1000
# The IP address of the machine's public network used by the mining program
ServiceAddr  = ""
# Port number monitored by the mining program
ServicePort  = 15001
# The address of income account
IncomeAcc    = ""
# phrase or seed of the signature account
SignaturePrk = ""
```

**<4> View bucket features**

`bucket` has many functions, you can use `-h` or `--help` to view, as follows:

- flag

| Flag        | Description                             |
| ----------- | --------------------------------------- |
| -c,--config | Custom profile |
| -h,--help   | Print help information                  |

- command

| Command  | Description                                    |
| -------- | ---------------------------------------------- |
| version  | Print version number                           |
| default  | Generate configuration file template           |
| register | Register mining miner information to the chain |
| state    | Query mining miner information                 |
| run      | Start mining normally                          |
| exit     | Exit the mining platform                       |
| increase | Increase the deposit of mining miner           |
| withdraw | Redemption deposit of mining miner             |

**<5> Use bucket**

**All `bucket` commands (except default and version) need to be registered before they can be used.**

```
sudo ./bucket register
```

- Query miner status

```
sudo ./bucket state
```

- Increase the miner's deposit by 1000

```
sudo ./bucket increase 1000
```

- Exit the mining platform

```
sudo ./bucket exit
```

- Redeem the miner's deposit

```
sudo ./bucket withdraw
```

- Start mining

```
sudo nohup ./bucket run > /dev/null 2>&1 &
```



## License
Licensed under [Apache 2.0](https://github.com/CESSProject/cess-bucket/blob/main/LICENSE)
