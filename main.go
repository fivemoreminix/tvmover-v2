package main

// TODO: JSON configuration
// Move internal code to private functions and other files
// Replace consts with JSON configuration
// Command line argument to specify configuration path

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	noConfiguredDirsMsg = "No configured directories. Exiting."
	firstRunMsg         = `A configuration file has been created at %s

Be sure to edit this file and configure the input and output directories, otherwise the program
will not run.

Use the log file to debug issues with your configuration. (path is specified in the config)

Note when writing your config: a '\' means that a special character follows. To get a single '\'
you must write two: '\\'. For this reason, it's easier to write paths on Windows using forward
slashes instead of backslashes: C:/Users/myuser/Documents rather than C:\\Users\\myuser\\Documents

You can delete this file.

Thanks for using TVMover.
Send issues or requests to <thelukaswils@gmail.com>`
)

var configPath = flag.String("config", "tvmover.json", "Specify an alternative config path")
var filenameRegexp *regexp.Regexp

func main() {
	flag.Parse()

	var firstRun bool

	if !strings.Contains(*configPath, string(os.PathSeparator)) { // Ex: "tvmover.json" not "abc/tvmover.json"
		*configPath = JoinPathToExeDir(*configPath)
	}

	// Find or create a config
	config := NewConfig()
	data, err := ioutil.ReadFile(*configPath)
	if os.IsNotExist(err) {
		// Make config if it does not exist
		marshalled, err := JSONMarshalIndent(config, "", "    ")
		if err != nil {
			log.Fatalf("Failed to interpret config: %v\n", err)
		}
		err = ioutil.WriteFile(*configPath, marshalled, 0666) // ✝
		if err != nil {
			log.Fatalf("Failed to create config file: %v\n", err)
		}

		firstRun = true
	} else {
		err = json.Unmarshal(data, config)
		if err != nil {
			log.Fatalf("Failed to interpret config: %v\n", err)
		}
	}

	// Make log relative to exe if it does not contain path separators
	logPathTmp := config.LogFile
	if !strings.Contains(config.LogFile, string(os.PathSeparator)) {
		logPathTmp = JoinPathToExeDir(config.LogFile)
	}

	// Log to file
	f, err := os.OpenFile(logPathTmp, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666) // ✝
	if err != nil {
		log.Fatalf("Failed to open or create the log file: %v\n", err)
	} else {
		defer f.Close()
		log.SetOutput(f)
	}

	if firstRun {
		readmePath := JoinPathToExeDir("readme.txt")
		fmt.Printf("Configuration created at '%s'\nCheck '%s'\n", *configPath, readmePath)
		ioutil.WriteFile(readmePath, []byte(fmt.Sprintf(firstRunMsg, *configPath)), 0666)
		log.Printf("First run: created '%s' and '%s'\n", *configPath, readmePath)
		return
	}

	if len(config.Dirs) == 0 {
		fmt.Println(noConfiguredDirsMsg)
		log.Fatalln(noConfiguredDirsMsg)
	}

	log.Println("Starting run...")

	filenameRegexp, err = regexp.Compile(config.FilenameRegex)
	if err != nil {
		log.Fatalf("Failed to compile FilenameRegex: %v", err)
	}

	var count int64
	for i, dir := range config.Dirs {
		log.Printf("Processing input directory %d: '%s'", i+1, dir.InDir)
		items := moveFromDir(dir.InDir, dir.OutDirs, config)
		log.Printf("Finished processing input directory %d, %d items.", i+1, items)

		count += items
	}

	var verbage string
	switch config.GetMoveMethod() {
	case Move:
		verbage = "moving"
	case Copy:
		verbage = "copying"
	}

	log.Printf("Finished %s %d items.", verbage, count)
}

type ConfigDir struct {
	InDir   string
	OutDirs []string
}

type Config struct {
	LogFile       string // Default = tvmover.log
	MoveMethod    string // TODO: implement
	FilenameRegex string
	Exts          []string    `json:"Extensions"`
	IgnoreChars   string      `json:"IgnoreCharacters"`
	Dirs          []ConfigDir `json:"Directories"`
}

func NewConfig() *Config {
	return &Config{
		"tvmover.log",
		"move",
		`(?P<title>.+)\b[Ss].*(?P<season>[0-9]+).*[Ee].*(?P<episode>[0-9]+).*`,
		[]string{".mp4", ".mov", ".avi"},
		",<>;:'\".-_+=(){}[]!@#$%^&*",
		[]ConfigDir{{InDir: "C:/Users/myuser/Documents/ExampleSrc", OutDirs: []string{"C:/Users/myuser/Documents/ExampleDest"}}},
	}
}

type MoveMethod uint8

const (
	Move MoveMethod = iota
	Copy
)

func (c *Config) GetMoveMethod() MoveMethod {
	switch strings.ToLower(c.MoveMethod) {
	case "move":
		return Move
	case "copy":
		return Copy
	default:
		log.Fatalf("MoveMethod is not 'move' or 'copy': %s", c.MoveMethod)
	}
	return 0
}

func JoinPathToExeDir(p string) string {
	exeDir, err := os.Executable()
	if err != nil { // *BSD?
		return p
	}
	return path.Join(path.Dir(exeDir), p)
}

func JSONMarshalIndent(t interface{}, prefix, indent string) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // The whole reason we use this function...
	encoder.SetIndent(prefix, indent)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}
