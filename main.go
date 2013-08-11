package main

import (
	"./androidsync"
	"log"
	"os"
)

func main() {
	synchro := androidsync.New()
	synchro.PathSeparator = "/"
	synchro.Logger = log.New(os.Stdin, "", log.LstdFlags)
	synchro.AdbPath = "/Developer/Applications/adt/sdk/platform-tools/adb"
	synchro.IgnorePattern("PSX.*")
	synchro.IgnorePattern("*.mkv")
	synchro.IgnorePattern("*.mp4")
	synchro.IgnorePattern("*.flv")
	synchro.IgnorePattern("*.iso")
	synchro.IgnorePattern("*.smc")
	synchro.IgnorePattern("cache/")
	synchro.IgnorePattern("Cache/")
	synchro.IgnorePattern("app_news_image_cache/")
	synchro.Synchronize("/", "/Volumes/Donnees/targetsync/")
}
