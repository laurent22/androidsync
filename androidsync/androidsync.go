package androidsync

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AndroSync struct {
	PathSeparator rune
	Logger *log.Logger
	AdbPath string
	ChangeInterval time.Duration
	ignoredPatterns []string
}

type FolderItem struct {
	path string
	name string
	isDirectory bool
	size int
	timestamp time.Time
	isReadable bool
}

func New() *AndroSync {
	output := new(AndroSync)
	output.PathSeparator = os.PathSeparator
	
	// Initialize to null logger
	null, err := os.Open(os.DevNull)
	if err != nil {
		// Can't see how this could fail, so panic
		panic("Cannot create null logger.")
	}
	output.Logger = log.New(null, "", 0)
	output.ChangeInterval = 2 * time.Second
	
	// Skip proc file system
	output.IgnorePattern("/proc/")
	// Skip other temp and virtual file systems
	output.IgnorePattern("/acct/")
	output.IgnorePattern("/dev/")
	output.IgnorePattern("/tmp/")
	output.IgnorePattern("/sys/")
	
	return output
}

func (this *AndroSync) IgnorePattern(pattern string) {
	this.ignoredPatterns = append(this.ignoredPatterns, pattern)
}

func (this *AndroSync) isIgnoredPath(path string) bool {
	for _, pattern := range this.ignoredPatterns {
		ok, _ := this.PatternMatchesFile(pattern, path)
		if ok {
			return true
		}
	}
	return false
}

func (this *AndroSync) GetFolderItems(folderPath string) ([]FolderItem, error) {
	var output []FolderItem
	var err error
	
	cmd := exec.Command(this.AdbPath, "shell", "ls", "-la", folderPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return output, errors.New(fmt.Sprint(err) + ": " + strings.TrimSpace(stderr.String()))
	}
	
	var matches [][]string
	dateRegex := regexp.MustCompile("\\s[\\d]{4}-[\\d]{2}-[\\d]{2}\\s[\\d]{2}:[\\d]{2}\\s")
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" { continue }
		matches = dateRegex.FindAllStringSubmatch(line, -1)
		if len(matches) < 1 || len(matches[0]) < 1 {
			return output, errors.New("Invalid file entry: " + line)
		}
		
		fileType := line[0]
		if fileType != '-' && fileType != 'd' {
			// Skip symbolic links and other special files.
			continue
		}
		
		isDirectory := fileType == 'd'
		
		// eg. drwxr-xr-x
		permissions := line[0:10]
		
		dateString := strings.TrimSpace(matches[0][0])
		timestamp, err := time.Parse("2006-01-02 15:04", dateString)
		if err != nil {
			return output, errors.New("Cannot parse date: " + dateString)
		}
		
		idx := strings.Index(line, dateString)
		idx += len(dateString) + 1
		filename := line[idx:len(line)]
		
		var fileSize int = 0
		if !isDirectory {
			idx = strings.Index(line, dateString)
			idx -= 2
			fileSizeString := ""
			allowedChars := "0123456789"
			for {
				c := line[idx]
				if strings.Index(allowedChars, string(c)) < 0 { break }
				fileSizeString = string(c) + fileSizeString
				idx--
			}
			
			fileSize, err = strconv.Atoi(fileSizeString)
			if err != nil {
				return output, errors.New("Cannot parse file size: " + line + ": " + fmt.Sprint(err)) 
			}
		}
		
		var folderItem FolderItem
		folderItem.name = filename
		folderItem.timestamp = timestamp
		folderItem.path = folderPath + filename
		if isDirectory { folderItem.path += string(this.PathSeparator) }
		folderItem.isDirectory = isDirectory
		folderItem.size = fileSize
		folderItem.isReadable = strings.Index(permissions, "r") > 0
		output = append(output, folderItem)
	}
	
	return output, nil
}

func (this *AndroSync) timestampAreSame(t1 time.Time, t2 time.Time, interval time.Duration) bool {
	d := t1.Sub(t2)
	if d < 0 { d = -d }
	if d <= interval {
		return true
	}
	// Hack: Mac OS X dates go back to 1980 at the earliest while Linux dates go
	// back to 1970. So if one of the date is before 1980 and another one in 1980,
	// we normalize the dates to 1980 and do a second comparison.
	if (t1.Year() == 1980 && t2.Year() < 1980) || (t1.Year() < 1980 && t2.Year() == 1980) {
		t1 = time.Date(1980, t1.Month(), t1.Day(), t1.Hour(), t1.Minute(), t1.Second(), t1.Nanosecond(), t1.Location())
		t2 = time.Date(1980, t2.Month(), t2.Day(), t2.Hour(), t2.Minute(), t2.Second(), t2.Nanosecond(), t2.Location())
		return this.timestampAreSame(t1, t2, interval)
	}
	return false
}

func (this *AndroSync) Synchronize(androidPath string, localPath string) error {
	if this.isIgnoredPath(androidPath) {
		// this.Logger.Println("Skipping: " + androidPath)
		return nil
	}
	
	var err error
	
	folderItems, err := this.GetFolderItems(androidPath)
	if err != nil {
		return err
	}
	
	err = os.MkdirAll(localPath, os.ModeDir | os.ModePerm)
	if err != nil {
		return err
	}
	
	for _, folderItem := range folderItems {
		if folderItem.isDirectory {
			err = this.Synchronize(folderItem.path, localPath + folderItem.name + string(this.PathSeparator))
			if err != nil {
				return err
			}
			continue
		} else {
			if this.isIgnoredPath(folderItem.path) {
				// this.Logger.Println("Skipping: " + folderItem.path)
				continue
			}
		}
		if !folderItem.isReadable {
			// Skip write-only files
			continue
		}
		targetPath := localPath + folderItem.name
		targetPathInfo, err := os.Stat(targetPath)
		fileHasChanged := false
		if !os.IsNotExist(err) {
			if err != nil {
				this.Logger.Println("ERROR: Could not get info on target path:", targetPath, ":", err)
				fileHasChanged = true
			} else {
				if int64(folderItem.size) == targetPathInfo.Size() {
					if !this.timestampAreSame(folderItem.timestamp, targetPathInfo.ModTime(), this.ChangeInterval) {
						fileHasChanged = true
					}
				} else {
					fileHasChanged = true
				}
			}
		} else {
			fileHasChanged = true
		}
		if !fileHasChanged {
			continue
		}
		cmd := exec.Command(this.AdbPath, "pull", folderItem.path, targetPath)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err = cmd.Run()
		stdoutString := strings.TrimSpace(stdout.String())
		stderrString := strings.TrimSpace(stderr.String())
		if err != nil {
			return errors.New(fmt.Sprint(err) + ": " + stderrString)
		}
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			this.Logger.Println("ERROR:", targetPath, " could not be copied.")
			continue
		}
		err = os.Chtimes(targetPath, time.Now(), folderItem.timestamp)
		if err != nil {
			this.Logger.Println("ERROR: could not set timestamp on", targetPath, ":", err)
		}
		logString := folderItem.path
		if stdoutString != "" { logString += ": " + stdoutString }
		if stderrString != "" { logString += ": " + stderrString }
		this.Logger.Println(logString)
	}
	
	return nil	
}

func (this *AndroSync) PatternMatchesFile(pattern string, filePath string) (bool, error) {	
	if len(pattern) == 0 || len(filePath) == 0 { return false, errors.New("Both pattern and file path must be specified.") }
	
	prependWildcard := false
	if pattern[0] == '*' {
		pattern = string(this.PathSeparator) + pattern
	} else if rune(pattern[0]) != this.PathSeparator {
		prependWildcard = true
	}
	
	pattern = strings.Replace(pattern, "\\", "\\\\", -1)
	pattern = strings.Replace(pattern, "[", "\\[", -1)
	pattern = strings.Replace(pattern, "]", "\\]", -1)
	pattern = strings.Replace(pattern, ".", "\\.", -1)
	pattern = strings.Replace(pattern, "*", ".+?", -1)
	if prependWildcard { pattern = ".*?" + string(this.PathSeparator) + pattern }
	pattern = "^" + pattern + "$"
	
	// TODO: compile and cache patterns?
	return regexp.MatchString(pattern, filePath)
}
