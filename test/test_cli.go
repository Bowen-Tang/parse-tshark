package main

import (
    "bufio"
    "fmt"
    "flag"
    "os"
    "strconv"
    "strings"
)

// QueryInfo 结构体用于存储查询信息
type QueryInfo struct {
    Sno   string
    Rt    float64
    Sip   string
    Sport string
    Sql   string
}

var (
    tsharkFile    string
)

func init() {
    flag.StringVar(&tsharkFile, "tshark", "tshark.log", "Path to the tshark log file")
    flag.Parse()
}


// processLine 函数用于处理每行数据
func processLine(fields []string, queries map[string]*QueryInfo) {
    if len(fields) < 7 {
        fmt.Println("Skipped a line due to insufficient fields:", strings.Join(fields, "|"))
        return
    }

    streamNo := fields[0]
    tcpLen, _ := strconv.Atoi(fields[1])
    timeDelta, _ := strconv.ParseFloat(fields[2], 64)
    srcIP := fields[3]
    srcPort := fields[4]
//    sql := strings.Join(fields[7:],"\\n")
    // 将 SQL 字段中的换行符替换为 \n
    sql := strings.Join(fields[7:], " ")
    sql = strings.ReplaceAll(sql, "\n", "\\n")

    if sql == "" {
        sql = "null"
    }

    if sql != "null" {
        // 如果 SQL 不为空，向 map 添加一行数据
        queries[streamNo] = &QueryInfo{
            Sno:   streamNo,
            Rt:    0,
            Sip:   srcIP,
            Sport: srcPort,
            Sql:   sql,
        }
    } else {
        // 如果 SQL 为空，检查 map 中是否存在该 streamNo
        if query, exists := queries[streamNo]; exists {
            query.Rt += timeDelta
            if tcpLen > 0 {
                // 打印信息并从 map 删除
                fmt.Printf("Stream_No: %s, Response_Time: %f, Source_IP: %s, Source_Port: %s, SQL: %s\n",
                    query.Sno, query.Rt, query.Sip, query.Sport, query.Sql)
                delete(queries, streamNo)
            }
        }
    }
}

func main() {
    // 打开文件
    file, err := os.Open(tsharkFile)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var currentFields []string
    queries := make(map[string]*QueryInfo)

    // 逐行读取和处理
    for scanner.Scan() {
        line := scanner.Text()
        fields := strings.Split(line, "|")

        if len(fields) >= 7 {
            // 如果之前有正在处理的行，先处理它
            if len(currentFields) > 0 {
                processLine(currentFields, queries)
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
        processLine(currentFields, queries)
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading file:", err)
    }
}
