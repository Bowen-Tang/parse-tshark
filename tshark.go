package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "hash/crc32"
    "os"
    "time"
    "strconv"
    "strings"
    "github.com/pingcap/tidb/pkg/parser"
)

// QueryInfo 结构体用于存储查询信息
type QueryInfo struct {
    Sno   string
    Rt    float64
    Sip   string
    Sport string
    Ts    float64 // 新增执行完成时间戳
    Sql   string
}

// HostInfo 结构体用于存储主机信息
type HostInfo struct {
    Host string `json:"host"`
    ID   int    `json:"id"`
    User string `json:"user"`
    DB   string `json:"db"`
}

// OutputEntry 结构体用于格式化输出信息
type OutputEntry struct {
    ConnectionID string `json:"connection_id"`
    QueryTime    int    `json:"query_time"`
    RowsSent     int    `json:"rows_sent"`
    Username     string `json:"username"`
    DBName       string `json:"dbname"`
    SQLType      string `json:"sql_type"`
    Digest       string `json:"digest"`
    Ts           float64 `json:"ts"` // 新增执行完成时间戳
    SQL          string `json:"sql"`
}

var rtValue float64

func ParseTshark(tsharkFile,hostInfoFile,replayoutFile,defaultUser,defaultDB,ParseMode string) {
    if tsharkFile == "" || hostInfoFile == "" || replayoutFile == "" || defaultUser == "" || defaultDB == "" {
        fmt.Println("Usage: ./parse-tshark -mode parse2file -parsemode 1 -tsharkfile ./tshark.log -hostfile host.ini -replayfile ./tshrark.out -defaultuser user_null -defaultdb db_null")
        return
    }
        // 读取 hostInfo 数据
    hostInfoMap := readHostInfo(hostInfoFile)

    // 打开 tshark 文件
    file, err := os.Open(tsharkFile)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    // 打开输出文件
    output, err := os.Create(replayoutFile)
    if err != nil {
        fmt.Println("Error creating output file:", err)
        return
    }
    defer output.Close()

    scanner := bufio.NewScanner(file)
    buf := make([]byte, 0, 512*1024*1024) // 512MB的缓冲区
    scanner.Buffer(buf, bufio.MaxScanTokenSize)
    var currentFields []string
    queries := make(map[string]*QueryInfo)

    // 逐行读取和处理
    for scanner.Scan() {
        line := scanner.Text()
        fields := strings.Split(line, "|")

        if len(fields) >= 8 {
            // 如果之前有正在处理的行，先处理它
            if len(currentFields) > 0 {
                processAndOutputLine(currentFields, queries, hostInfoMap, output,defaultUser ,defaultDB,ParseMode)
                currentFields = []string{}
            }
            currentFields = fields
        } else {
            // 继续收集跨行的 SQL 语句
            currentFields = append(currentFields, "\n"+line)
        }
    }

    // 处理最后一行
    if len(currentFields) > 0 {
        processAndOutputLine(currentFields, queries, hostInfoMap, output,defaultUser ,defaultUser,ParseMode)
    }

    // 处理 map 中剩余的 queries
    for _, query := range queries {
        currentTimestamp := float64(time.Now().UnixNano()) / 1e9
        query.Rt = currentTimestamp - query.Ts // 计算 QueryTime
        outputEntry := createOutputEntry(query, hostInfoMap, query.Sip+":"+query.Sport, defaultUser, defaultDB)
        jsonData, _ := json.Marshal(outputEntry)
        output.WriteString(string(jsonData) + "\n")
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading file:", err)
    }
}

func processAndOutputLine(fields []string, queries map[string]*QueryInfo, hostInfoMap map[string]HostInfo, output *os.File,defaultUser ,defaultDB,ParseMode string) {
    if len(fields) < 8 {
        fmt.Println("Skipped a line due to insufficient fields:", strings.Join(fields, "|"))
        return
    }

    streamNo := fields[0]
    tcpLen, _ := strconv.Atoi(fields[1])
    timeDelta, _ := strconv.ParseFloat(fields[2], 64)
    srcIP := fields[3]
    srcPort := fields[4]
    ts, _ := strconv.ParseFloat(fields[7], 64)
    sql := strings.Join(fields[8:], " ")

    if sql == "" {
        sql = "null"
    }

    if sql != "null" {
        if ParseMode == "1" {
            rtValue = 0
        } else if ParseMode == "2" {
            rtValue = timeDelta
        }

        // 如果 SQL 不为空，向 map 添加一行数据
        queries[streamNo] = &QueryInfo{
            Sno:   streamNo,
            Rt:    rtValue,
            Sip:   srcIP,
            Sport: srcPort,
            Ts:    ts, // 增加执行完成时间戳
            Sql:   sql,
        }
    } else if query, exists := queries[streamNo]; exists {
        if ParseMode == "1" {
            query.Rt += timeDelta
            if tcpLen > 0 {
                // 将信息写入输出文件
                outputEntry := createOutputEntry(query, hostInfoMap, srcIP+":"+srcPort, defaultUser, defaultDB)
                jsonData, _ := json.Marshal(outputEntry)
                output.WriteString(string(jsonData) + "\n")

                // 从 map 删除
                delete(queries, streamNo)
            }
        } else if ParseMode == "2" {
            if tcpLen > 0 {
                query.Rt = timeDelta - query.Rt // 更新 Rt
                // 将信息写入输出文件
                outputEntry := createOutputEntry(query, hostInfoMap, srcIP+":"+srcPort, defaultUser, defaultDB)
                jsonData, _ := json.Marshal(outputEntry)
                output.WriteString(string(jsonData) + "\n")

                // 从 map 删除
                delete(queries, streamNo)
            }
        }
    }
}

func createOutputEntry(query *QueryInfo, hostInfoMap map[string]HostInfo, host ,defaultUser , defaultDB string) OutputEntry {
    // 构建完整的 host 字符串
    completeHost := query.Sip + ":" + query.Sport

    var connectionID string
    var username string
    var dbName string

    // 尝试从 hostInfoMap 中找到对应的 HostInfo
    if info, exists := hostInfoMap[completeHost]; exists {
        connectionID = fmt.Sprintf("%d", info.ID)
        username = info.User
        if username == "" {
            username = defaultUser
        }
        dbName = info.DB
        if dbName == "" || dbName == "null" { // 检查是否为 "null" 并替换为 defaultDB
            dbName = defaultDB
        }
    } else {
        // 如果 hostInfoMap 中没有匹配项，则使用 crc32 值作为 connectionID
        crc32ID := crc32.ChecksumIEEE([]byte(completeHost))
        connectionID = fmt.Sprintf("%d", crc32ID)
        username = defaultUser
        dbName = defaultDB
    }

    sqlType,sqlDigest := getSQLInfo(query.Sql)

    return OutputEntry{
        ConnectionID: connectionID,
        QueryTime:    int(query.Rt * 1000000),
        RowsSent:     0,
        Username:     username,
        DBName:       dbName,
        SQLType:      sqlType,
        Digest:       sqlDigest,
        Ts:           query.Ts, // 增加执行完成时间戳
        SQL:          query.Sql,
    }
}

func readHostInfo(filename string) map[string]HostInfo {
    file, err := os.Open(filename)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return nil
    }
    defer file.Close()

    hostInfoMap := make(map[string]HostInfo)
    scanner := bufio.NewScanner(file)
    buf := make([]byte, 0, 512*1024*1024) // 512MB的缓冲区
    scanner.Buffer(buf, bufio.MaxScanTokenSize)
    for scanner.Scan() {
        var info HostInfo
        json.Unmarshal([]byte(scanner.Text()), &info)
        hostInfoMap[info.Host] = info
    }
    return hostInfoMap
}

func getSQLInfo(sql string) (string, string) {
    normalizedSQL := parser.Normalize(sql)
    digest := parser.DigestNormalized(normalizedSQL).String()
    words := strings.Fields(normalizedSQL) 

    if len(words) > 0 {
        return words[0], digest
    }
    
    return "other", digest
}
