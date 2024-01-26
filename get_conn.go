package main

import (
        "database/sql"
        "encoding/json"
        "flag"
        "fmt"
        "os"
        "time"

        _ "github.com/go-sql-driver/mysql"
)

type ProcessInfo struct {
        Host string        `json:"host"`
        ID   int           `json:"id"`
        User string        `json:"user"`
        DB   sql.NullString
}

// DBString 返回 DB 字段的字符串表示。
func (pi ProcessInfo) DBString() string {
        if pi.DB.Valid {
                return pi.DB.String
        }
        return "null" // 或者可以返回空字符串 ""，取决于您的需求
}

var (
        dbInfo     string
        outputFile string
)

func init() {
        flag.StringVar(&dbInfo, "dbinfo", "username:password@tcp(localhost:3306)/", "Database connection information")
        flag.StringVar(&outputFile, "output", "output.json", "Output file name")
}

func main() {
        flag.Parse()

        db, err := sql.Open("mysql", dbInfo)
        if err != nil {
                panic(err)
        }
        defer db.Close()

        existingRecords := make(map[int]bool)

        for {
                processList, err := queryDatabase(db)
                if err != nil {
                        fmt.Println("Error querying database:", err)
                        continue
                }

                writeToFile(processList, outputFile, existingRecords)
                time.Sleep(500 * time.Millisecond)
        }
}

func queryDatabase(db *sql.DB) ([]ProcessInfo, error) {
        query := "SELECT host, id, user, db FROM information_schema.processlist WHERE user <> 'event_scheduler'"
        rows, err := db.Query(query)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var processList []ProcessInfo
        for rows.Next() {
                var pi ProcessInfo
                if err := rows.Scan(&pi.Host, &pi.ID, &pi.User, &pi.DB); err != nil {
                        return nil, err
                }
                processList = append(processList, pi)
        }
        return processList, nil
}

func writeToFile(processList []ProcessInfo, filePath string, existingRecords map[int]bool) {
        if len(processList) == 0 {
                return
        }

        file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
                fmt.Println("Error opening file:", err)
                return
        }
        defer file.Close()

        for _, pi := range processList {
                if _, exists := existingRecords[pi.ID]; exists {
                        continue
                }
                existingRecords[pi.ID] = true

                // 使用自定义的 DBString 方法
                output := struct {
                        Host string `json:"host"`
                        ID   int    `json:"id"`
                        User string `json:"user"`
                        DB   string `json:"db"`
                }{
                        Host: pi.Host,
                        ID:   pi.ID,
                        User: pi.User,
                        DB:   pi.DBString(),
                }

                jsonData, err := json.Marshal(output)
                if err != nil {
                        fmt.Println("Error marshalling JSON:", err)
                        continue
                }
                file.Write(jsonData)
                file.WriteString("\n")
        }
}
