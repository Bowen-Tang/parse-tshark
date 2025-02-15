# 工具说明
解析 tshark 工具生成的 MySQL SQL、解析响应时间（响应时间按照 SQL 第一次返回数据包计算，不考虑应用流式读取数据时发送结果的时间）

tshark 需提前安装：

```
yum install -y wireshark # Centos 7 自带的版本较低，但也能工作，建议编译安装 3.2.3 版本
```


# 使用说明
## 1. 使用 tshark 抓取 MySQL 数据包（tcpdump 抓取的数据包 parse-tshark 工具无法正确处理）
### 方式一：使用 tshark 进行 port+mysql 过滤（不推荐）
该方式会直接生成 parse-tshark 工具可读取的文件，生成的文件比较小，但在资源不够时对 MySQL 性能影响大（不推荐在生产使用）
```
sudo tshark -i eth0 -Y "mysql.query or ( tcp.srcport==3306)" -d tcp.port==3306,mysql -o tcp.calculate_timestamps:true -T fields -e tcp.stream -e tcp.len -e tcp.time_delta -e ip.src -e tcp.srcport -e ip.dst -e tcp.dstport -e frame.time_epoch -e mysql.query -E separator='|' >> tshark.log
```
### 方式二：使用 tshark 进行 port 过滤，再二次过滤文件中的 mysql.query 和 响应时间
该方式生成的文件比较大，但对生产性能影响小
#### 抓包
该命令只是根据 3306 端口和 eth0 网卡抓包，以抓取 1 小时为例（每个文件大约 2000MB，最多生成 200 个）
```
sudo tshark -i eth0 -f "tcp port 3306" -a duration:3600 -b filesize:2000000 -b files:200 -w ts.pcap
```
#### 分析包
该命令针对**抓包**生成的 pcap 文件进行处理，处理成 parst-tshark 工具可读的文件（建议将这些文件传输到回放服务器处理）
```
for i in `ls -rth ts*.pcap`
do
sudo tshark -r $i -Y "mysql.query or ( tcp.srcport==3306)" -d tcp.port==3306,mysql -o tcp.calculate_timestamps:true -T fields -e tcp.stream -e tcp.len -e tcp.time_delta -e ip.src -e tcp.srcport -e ip.dst -e tcp.dstport -e frame.time_epoch -e mysql.query -E separator='|' >> tshark.log
done
```
## 2. 获取抓包过程中的 user db 信息
由于 tshark 抓包时获取 user/db 信息过于复杂、且存在局限性，所以通过工具每隔 500ms 获取一次 MySQL 数据库的 processlist 视图信息，通过源端 IP+端口 与 processlist 视图中的 host 匹配

```
./parse-tshark -mode getmysql -dbinfo 'username:password@tcp(localhost:3306)/information_schema' -output host.ini
```
注意：该工具运行时间需要和 tshark 抓包时间一样久，才能获取完整的 user/db 信息

如抓取的是 mycat 中间件流量，则需要使用如下命令：

```
./parse-tshark -mode getmycat -dbinfo 'username:password@tcp(localhost:9066)' -output host.ini

```

注意：mycat show @@connection 默认没记录 user 信息，所以在 host.ini 中显示的是 null

## 3. 解析数据包
### 3.1 打印模式：将数据包中的 SQL 信息等打印到屏幕（该模式仅适用于调试）

```
./parse-tshark -mode parse2cli -parsemode 1 -tsharkfile ./tshark.log
```
### 3.2 解析模式：生成 sql-replay 可回放的文件
```
./parse-tshark -mode parse2file -parsemode 1 -tsharkfile ./tshark.log -hostfile host.ini -replayfile ./tshark.out -defaultuser user_null -defaultdb db_null
```
## 4. 使用 sql-replay 进行回放
说明：sql-replay 是一个回放 MySQL 慢查询日志的工具：[sql-replay](https://github.com/Bowen-Tang/sql-replay)

# 抓包对性能的影响
1. 测试环境: 8C VM
2. MySQL: 8.0.33
3. tshark: 3.2.3
4. 测试用例: sysbench

| 并发 | 初始 TPS | CPU   | TPS: tshark port | TPS: tshark port+mysql |TPS: tcpdump port |
|:----:|:--------:|:-----:|:-----------:|:-----------------:|:------------:|
| 1    | 148.28   | 25.6% | 143.56      | 138.20            | 145.14       |
| 5    | 342.12   | 37.9% | 324.85      | 320.24            | 326.13       |
| 10   | 525.76   | 47.7% | 495.98      | 457.25            | 511.26       |
| 50   | 1103.53  | 73.9% | 1017.46     | 871.50            | 1045.98      |
| 100  | 1301.19  | 79.8% | 1237.04     | 968.46            | 1255.13      |

**说明**
1. 低并发+资源充足时，“tshark 抓包方式一”、“tshark 抓包方式二”、“tcpdump 抓包方式”三者对 MySQL 的影响均不大
2. 高并发+资源不够时，“tshark 抓包方式一”有 7% 影响，“tshark 抓包方式二”有 21% 影响，“tcpdump 抓包方式”有 5% 影响

# 抓包成功率
sysbench 50 并发 * select_random_points * 1800 秒，总共执行 45757727 条 SQL

![image](https://github.com/Bowen-Tang/parse-tshark/assets/52245161/caeacc29-bdde-43b6-ac4e-5350603832a7)

parse-tshark 最终解析出 45747497 条 SQL

![1706367141839](https://github.com/Bowen-Tang/parse-tshark/assets/52245161/6eb5ed7b-078a-4459-b62d-88363f856054)


# 已知问题
1. 负载均衡节点抓包时，无法根据 host:port 来获得 id,user,db 等信息，只能在 parse2file 时使用 -defaultuser -defaultdb 手工指定
2. mycat 节点抓包时，无法获取到 user 信息，只能在 parse2file 时 -defaultuser 手工指定
3. 上游是 prepare 预编译 SQL 时，解析出来的 SQL 无法执行，在 sql-replay 时这部分 SQL 在数据库中会报错
4. 抓包开始到抓包过程结束还没执行完成的 SQL，不会解析到输出文件中

[tshark MySQL 过滤器说明](https://www.wireshark.org/docs/dfref/m/mysql.html#:~:text=Protocol%20field%20name:%20mysql.%20Versions:%201.0.0%20to%204.4.0.%20Back%20to)

# 感谢[@plantegg](https://plantegg.github.io/)大佬分享的抓包方法
[就是要你懂抓包](https://plantegg.github.io/2019/06/21/%E5%B0%B1%E6%98%AF%E8%A6%81%E4%BD%A0%E6%87%82%E6%8A%93%E5%8C%85--WireShark%E4%B9%8B%E5%91%BD%E4%BB%A4%E8%A1%8C%E7%89%88tshark/)
![image](https://github.com/Bowen-Tang/parse-tshark/assets/52245161/c1f28317-c5c6-43bb-b568-3ce9eb7504a3)
# 感谢[@zr-hebo](https://github.com/zr-hebo/sniffer-agent)大佬提供的长连接获取连接信息的思路
