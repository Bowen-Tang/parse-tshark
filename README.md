# parse-tshark
解析 tshark 数据包中的 SQL


sudo tshark -Y "mysql.query or (  tcp.srcport==3306)" -o tcp.calculate_timestamps:true -T fields -e tcp.stream -e tcp.len -e tcp.time_delta -e ip.src -e tcp.srcport -e ip.dst -e tcp.dstport -e mysql.query -E separator='|'
