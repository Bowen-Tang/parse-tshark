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

type ConnectionInfo struct {
        Host      string
        LocalPort int
        ID        int
        Schema    sql.NullString
}

func (ci ConnectionInfo) JSONOutput() (string, error) {
        output := struct {
                Host string `json:"host"`
                ID   int    `json:"id"`
                User string `json:"user"`
                DB   string `json:"db"`
        }{
                Host: fmt.Sprintf("%s:%d", ci.Host, ci.LocalPort),
                ID:   ci.ID,
                User: "null",
                DB:   ci.schemaString(),
        }

        jsonData, err := json.Marshal(output)
        if err != nil {
                return "", err
        }

        return string(jsonData), nil
}

func (ci ConnectionInfo) schemaString() string {
        if ci.Schema.Valid {
                return ci.Schema.String
        }
        return "null"
}

var (
        dbInfo     string
        outputFile string
)

func init() {
        flag.StringVar(&dbInfo, "dbinfo", "username:password@tcp(localhost:3306)/", "Database connection information")
        flag.StringVar(&outputFile, "output", "host.ini", "Output file name")
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
                connectionList, err := queryDatabase(db)
                if err != nil {
                        fmt.Println("Error querying database:", err)
                        continue
                }

                writeToFile(connectionList, outputFile, existingRecords)
                time.Sleep(500 * time.Millisecond)
        }
}

func queryDatabase(db *sql.DB) ([]ConnectionInfo, error) {
        query := "show @@connection"
        rows, err := db.Query(query)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var connectionList []ConnectionInfo
        for rows.Next() {
                var (
                        ci           ConnectionInfo
                        processor    string
                        port         int
                        charset      string
                        netIn        int
                        netOut       int
                        activeTime   string
                        recvBuffer   int
                        sendQueue    int
                        txlevel      string
                        autocommit   string
                        sqlStatement string
                )
                if err := rows.Scan(&processor, &ci.ID, &ci.Host, &port, &ci.LocalPort, &ci.Schema, &charset, &netIn, &netOut, &activeTime, &recvBuffer, &sendQueue, &txlevel, &autocommit, &sqlStatement); err != nil {
                        return nil, err
                }
                connectionList = append(connectionList, ci)
        }
        return connectionList, nil
}

func writeToFile(connectionList []ConnectionInfo, filePath string, existingRecords map[int]bool) {
        if len(connectionList) == 0 {
                return
        }

        file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
                fmt.Println("Error opening file:", err)
                return
        }
        defer file.Close()

        for _, ci := range connectionList {
                if _, exists := existingRecords[ci.ID]; exists {
                        continue
                }
                existingRecords[ci.ID] = true

                jsonOutput, err := ci.JSONOutput()
                if err != nil {
                        fmt.Println("Error marshalling JSON:", err)
                        continue
                }
                file.WriteString(jsonOutput + "\n")
        }
}
