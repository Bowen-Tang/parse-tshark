package main

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
)

// QueryInfoc1 结构体用于存储查询信息
type QueryInfoc1 struct {
    Sno   string
    Rt    float64
    Sip   string
    Sport string
    Sql   string
}




// processLine1 函数用于处理每行数据
func processLine1(fields []string, queries map[string]*QueryInfoc1, ParseMode string) {
    if len(fields) < 8 {
        fmt.Println("Skipped a line due to insufficient fields:", strings.Join(fields, "|"))
        return
    }

    streamNo := fields[0]
    tcpLen, _ := strconv.Atoi(fields[1])
    timeDelta, _ := strconv.ParseFloat(fields[2], 64)
    srcIP := fields[3]
    srcPort := fields[4]
    sql := strings.Join(fields[8:], " ")
    sql = strings.ReplaceAll(sql, "\n", "\\n")

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
        queries[streamNo] = &QueryInfoc1{
            Sno:   streamNo,
            Rt:    rtValue,
            Sip:   srcIP,
            Sport: srcPort,
            Sql:   sql,
        }
    } else {
        if query, exists := queries[streamNo]; exists {
            if ParseMode == "1" {
                query.Rt += timeDelta
                if tcpLen > 0 {
                    // 打印信息并从 map 删除
                    fmt.Printf("Stream_No: %s, Response_Time: %f, Source_IP: %s, Source_Port: %s, SQL: %s\n",
                        query.Sno, query.Rt, query.Sip, query.Sport, query.Sql)
                    delete(queries, streamNo)
                }
            } else if ParseMode == "2" {
                if tcpLen > 0 {
                    query.Rt = timeDelta - query.Rt // 更新 Rt
                    // 将信息写入输出文件
                    fmt.Printf("Stream_No: %s, Response_Time: %f, Source_IP: %s, Source_Port: %s, SQL: %s\n",
                        query.Sno, query.Rt, query.Sip, query.Sport, query.Sql)
                    delete(queries, streamNo)
                }
            }
        }
    }
}

// func main() {
func Cli(tsharkFile ,ParseMode string) {
    if tsharkFile == "" {
        fmt.Println("Usage: ./parse-tshark -mode parse2cli -parsemode 1 -tsharkfile ./tshark.log")
        return
    }

    // 打开文件
    file, err := os.Open(tsharkFile)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var currentFields []string
    queries := make(map[string]*QueryInfoc1)

    // 逐行读取和处理
    for scanner.Scan() {
        line := scanner.Text()
        fields := strings.Split(line, "|")

        if len(fields) >= 8 {
            // 如果之前有正在处理的行，先处理它
            if len(currentFields) > 0 {
                processLine1(currentFields, queries,ParseMode)
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
        processLine1(currentFields, queries,ParseMode)
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading file:", err)
    }
}
