package main

import (
    "flag"
    "fmt"
    "os"
)

var version = "0.1.3, build date 20241011"

var showVersion bool

func main() {
    var mode string
    flag.StringVar(&mode, "mode", "", "Mode of operation: parse2cli ,parse2file , getmysql, or getmycat")

    // 共用的标志
    var ParseMode string
    flag.BoolVar(&showVersion, "version", false, "Show version info")
    var tsharkFile, dbInfo, outputFile,hostInfoFile , replayoutFile, defaultUser, defaultDB string
    flag.StringVar(&ParseMode, "parsemode", "1", "tshark capture mode, 1 or 2")
    flag.StringVar(&tsharkFile, "tsharkfile", "", "Path to the tshark log file: tshark.log")
    flag.StringVar(&dbInfo, "dbinfo", "", "Database connection information: username:password@tcp(localhost:3306)/db")
    flag.StringVar(&outputFile, "output", "", "Output file name: host.ini")
    flag.StringVar(&hostInfoFile, "hostfile", "", "Output file name: host.ini")
    flag.StringVar(&replayoutFile, "replayfile", "", "Replay(formated) Output file name: tshark.out")
    flag.StringVar(&defaultUser, "defaultuser", "", "Default username if not provided: user_null")
    flag.StringVar(&defaultDB, "defaultdb", "", "Default database if not provided: db_null")
    flag.Parse()

    if showVersion {
        fmt.Println("SQL Replay Tool Version:", version)
        os.Exit(0)
    }


    if mode == "" {
        fmt.Println("Usage: ./parse-tshark -mode [parse2cli|parse2file|getmysql|getmycat] -...")
        os.Exit(1)
    }

    switch mode {
    case "parse2cli":
        Cli(tsharkFile,ParseMode)
    case "parse2file":
        ParseTshark(tsharkFile,hostInfoFile,replayoutFile,defaultUser,defaultDB,ParseMode)
    case "getmysql":
        Get_Mysql(dbInfo,outputFile)
    case "getmycat":
        Get_Mycat(dbInfo,outputFile)
    default:
        fmt.Println("Invalid mode. Available modes: parse2cli|parse2file|getmysql|getmycat")
        os.Exit(1)
    }
}
