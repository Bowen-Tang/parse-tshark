# 工具说明
解析 tshark 工具生成的 MySQL SQL、解析响应时间

tshark 需提前安装：

```
yum install -y wireshark # Centos 7 自带的版本较低，但也能工作，建议编译安装 3.2.3 版本
```


# 使用说明
## 1. 使用 tshark 抓取 MySQL 数据包

```
sudo tshark -Y "mysql.query or ( tcp.srcport==3306)" -o tcp.calculate_timestamps:true -T fields -e tcp.stream -e tcp.len -e tcp.time_delta -e ip.src -e tcp.srcport -e ip.dst -e tcp.dstport -e mysql.query -E separator='|'
```
注意：一定要使用该命令才能生成该工具能够解析的格式
## 2. 获取抓包过程中的 user db 信息
由于抓包时抓取 user/db 信息过于复杂，所以通过工具每隔 500ms 获取一次 MySQL 数据库的 processlist 视图信息

```
./parse-tshark -mode getmysql -dbinfo 'username:password@tcp(localhost:3306)/information_schema' -output host.ini
```
注意：该工具运行时间需要和 tshark 抓包时间一样久，才能获取完整的 user/db 信息

如抓取的是 mycat 中间件流量，则需要使用如下命令：

```
./parse-tshark -mode getmycat -dbinfo 'username:password@tcp(localhost:9066)' -output host.ini

```
mycat show @@connection 默认没记录 user 信息，所以抓出来是 null
## 3. 解析数据包
仅打印

```
./parse-tshark -mode parse2cli -tsharkfile ./tshark.log
```
生成 sql-replay 可回放的文件

```
./parse-tshark -mode parse2file -tsharkfile ./tshark.log -hostfile host.ini -replayfile ./tshrark.out -defaultuser user_null -defaultdb db_null

```
## 4. 使用 sql-replay 进行回放
说明：sql-replay 默认是一个回放 MySQL 慢查询日志的工具：https://github.com/Bowen-Tang/sql-replay
