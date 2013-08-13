package main

import (
	"./androidsync"
	"os"
	"log"
	"flag"
	"strings"
	"fmt"
)

func Usage(errorMessage string) {
	if errorMessage != "" {
		fmt.Println("Error:", errorMessage)
	}
	fmt.Println("Usage: androidsync [flags] [source folder] [target folder]")
	fmt.Println("")
	flag.PrintDefaults()
}

func main() {
	var flagAdb string
	var flagExclude string
	var flagHelp bool
	
	flag.StringVar(&flagAdb, "adb", "", "Full path of adb executable.")
	flag.StringVar(&flagExclude, "exclude", "", "File or folder paths to exclude, separated by ';'. Wildcard '*' supported. End path with a slash to specify a folder.")
	flag.BoolVar(&flagHelp, "help", false, "Displays help message.")
	flag.Parse()
	
	flagAdb = strings.TrimSpace(flagAdb)
	flagExclude = strings.TrimSpace(flagExclude)
		
	if flagHelp {
		Usage("")
		return
	}
	
	if flagAdb == "" {
		Usage("adb path not specified.")
		return
	}
	
	args := flag.Args()
	if len(args) != 2 {
		Usage("source and target path not specified.")
		return
	}
	
	sourcePath := args[0]
	targetPath := args[1]
	
	synchro := androidsync.New()
	
	if flagExclude != "" {
		excludedItems := strings.Split(flagExclude, ";")
		for _, p := range excludedItems {
			synchro.IgnorePattern(strings.TrimSpace(p))
		}
	}
	
	// TODO: Check source and target paths have a trailing "/"
	
	synchro.Logger = log.New(os.Stdin, "", log.LstdFlags)
	synchro.AdbPath = flagAdb
	synchro.Synchronize(sourcePath, targetPath)
}