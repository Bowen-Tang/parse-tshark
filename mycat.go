package main

import (
        "database/sql"
        "encoding/json"
        "fmt"
        "os"
        "time"

        _ "github.com/go-sql-driver/mysql"
)

type ConnectionInfo1 struct {
        Host      string
        LocalPort int
        ID        int
        Schema    sql.NullString
}

func (ci ConnectionInfo1) JSONOutput() (string, error) {
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

func (ci ConnectionInfo1) schemaString() string {
        if ci.Schema.Valid {
                return ci.Schema.String
        }
        return "null"
}



func Get_Mycat(dbInfo,outputFile string) {
    if dbInfo == "" || outputFile == "" {
        fmt.Println("Usage: ./parse-tshark -mode getmycat -dbinfo 'username:password@tcp(localhost:9066)' -output host.ini")
        return
    }

        db, err := sql.Open("mysql", dbInfo)
        if err != nil {
                panic(err)
        }
        defer db.Close()

        existingRecords := make(map[int]bool)

        for {
                connectionList, err := queryDatabase1(db)
                if err != nil {
                        fmt.Println("Error querying database:", err)
                        continue
                }

                writeToFile1(connectionList, outputFile, existingRecords)
                time.Sleep(500 * time.Millisecond)
        }
}

func queryDatabase1(db *sql.DB) ([]ConnectionInfo1, error) {
        query := "show @@connection"
        rows, err := db.Query(query)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var connectionList []ConnectionInfo1
        for rows.Next() {
                var (
                        ci           ConnectionInfo1
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

func writeToFile1(connectionList []ConnectionInfo1, filePath string, existingRecords map[int]bool) {
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
