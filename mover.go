package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

func moveFromDir(src string, dsts []string, config *Config) (count int64) {
	fileinfos, err := ioutil.ReadDir(src)
	if err != nil {
		log.Fatalln(err)
	}

	deleteSources := config.GetMoveMethod() == Move

	for _, entry := range fileinfos {
		if entry.IsDir() { // If this entry in the recents directory is a folder...
			dirPath := path.Join(src, entry.Name())

			files, err := ioutil.ReadDir(dirPath)
			if err != nil {
				log.Println(err)
				continue
			}
			for _, file := range files {
				if FilenameHasExtension(file.Name(), config.Exts) {
					moveFileToDirs(path.Join(dirPath, file.Name()), dsts, deleteSources, config)
					count++
				}
			}

			// Delete show folder from recents when we're done copying the video files
			log.Printf("Finished sorting files from subdirectory '%s'", dirPath)
			if deleteSources { // Only remove origin directories when asked to move files
				log.Printf("Removing subdirectory '%s'", dirPath)
				err = os.RemoveAll(dirPath)
				if err != nil {
					log.Println(err)
				}
			}
		} else {
			if FilenameHasExtension(entry.Name(), config.Exts) {
				filePath := path.Join(src, entry.Name())
				moveFileToDirs(filePath, dsts, deleteSources, config)
				count++
			}
		}
	}
	return count
}

func moveFileToDirs(src string, dstBaseDirs []string, move bool, config *Config) {
	meta := NewFileMeta(path.Base(src), filenameRegexp, config.IgnoreChars) // Get filename "FileMeta" on the source file

	for i, dst := range dstBaseDirs {
		// attempt to find existing folder in TVDir that matches show title
		showPath := getTargetShowDirectory(dst, meta.CleanedTitle, meta)

		seasonDirTarget := fmt.Sprintf("Season %d", meta.Season)

		seasonDirs, err := ioutil.ReadDir(showPath)
		if err != nil {
			log.Println(err)
		}
		seasonDirExists := false
		for _, seasonDir := range seasonDirs {
			if seasonDir.IsDir() && seasonDir.Name() == seasonDirTarget {
				seasonDirExists = true
				break
			}
		}
		if !seasonDirExists {
			os.Mkdir(path.Join(showPath, seasonDirTarget), os.ModePerm)
		}

		// metaShowTitle = strings.Join(strings.Split(metaShowTitle, " "), ".")
		// if meta.Year != 0 { // Append production year to title if one is found
		// 	metaShowTitle = fmt.Sprintf(metaShowTitle, "%s.%s", metaShowTitle, meta.Year)
		// }
		// newFileName := fmt.Sprintf("%s.Season.%d.Episode.%d%s", metaShowTitle, meta.Season, meta.Episode, filepath.Ext(srcFile.Name()))

		outFilePath := path.Join(showPath, seasonDirTarget, meta.OriginalFilename)

		if move && i == len(dstBaseDirs)-1 { // If we're meant to move not copy and this is the last item
			os.Rename(src, outFilePath)
			log.Printf("Moved   '%s'  ->  '%s'", src, outFilePath)
		} else {
			copy(src, outFilePath)
			log.Printf("Copied  '%s'  ->  '%s'", src, outFilePath)
		}
	}
}

func getTargetShowDirectory(outputDir, showTitle string, meta *FileMeta) string {
	files, err := ioutil.ReadDir(outputDir)
	if err != nil {
		log.Println(err)
		log.Fatalln("Exiting due to previous error to prevent accidentally deleting unmoved files.")
	}
	showDirExists := false
	var tvShowDirPath string
	for _, showDir := range files {
		if showDir.IsDir() && strings.HasPrefix(strings.ToLower(showDir.Name()), strings.ToLower(showTitle)) { // Folder exists containing show title...
			if year := YearRegexp.FindString(showDir.Name()); year != "" { // If the show folder has a year in its name...
				year, _ := strconv.Atoi(year)
				if meta.Year != year { // If that year is not equal to a year found in the file meta...
					continue // Skip this show folder
				}
			}
			showDirExists = true
			tvShowDirPath = path.Join(outputDir, showDir.Name())
			break
		}
	}

	if !showDirExists { // If the show directory does not exist in the output...
		tvShowDirPath = path.Join(outputDir, showTitle) // Use calculated show title's casing
		os.Mkdir(tvShowDirPath, os.ModePerm)            // Create the show directory
	}

	return tvShowDirPath
}

func FilenameHasExtension(name string, extensions []string) bool {
	ext := strings.ToLower(path.Ext(name))
	for i := range extensions {
		if strings.ToLower(extensions[i]) == ext {
			return true
		}
	}
	return false
}

func ShowTagsMatchShowString(showTags []string, showTitle string) bool {
	joined := strings.Join(showTags, " ")
	return joined == showTitle
}

type FileMeta struct {
	OriginalFilename string
	CleanedTitle     string // "Doctor Who" from "doctor.who"
	Year             int
	Season           int
	Episode          int
}

//var FilenameRegexp = regexp.MustCompile(`[sS]\w*\s*(?P<season>[0-9]+)\s*[eE]\w*\s*(?P<episode>[0-9]+)`)
// var FilenameRegexp = regexp.MustCompile(`(?P<title>.+)\b[Ss].*(?P<season>[0-9]+).*[Ee].*(?P<episode>[0-9]+).*`)
// var IgnoreChars = ".-_"
var YearRegexp = regexp.MustCompile(`[0-9]{4}`)

func NewFileMeta(fileBaseName string, regexp *regexp.Regexp, ignoreChars string) *FileMeta {
	meta := &FileMeta{}
	meta.OriginalFilename = fileBaseName

	match := regexp.FindStringSubmatch(fileBaseName)
	captures := make(map[string]string)
	for i, name := range regexp.SubexpNames() {
		if i != 0 && name != "" {
			captures[name] = match[i]
		}
	}

	meta.CleanedTitle = CleanTitle(captures["title"], ignoreChars)
	season, _ := strconv.Atoi(captures["season"])
	meta.Season = season
	episode, _ := strconv.Atoi(captures["episode"])
	meta.Episode = episode

	// words := strings.FieldsFunc(filename, isSeparator)
	// for _, word := range words {
	// 	if YearRegexp.MatchString(word) {
	// 		meta.Year, _ = strconv.Atoi(word)
	// 	} else if values := SeasonEpisodeRegexp.FindAllStringSubmatch(word, 1); len(values) != 0 {
	// 		// values could be [[S01E03 01 03]] or []
	// 		season, _ := strconv.Atoi(values[0][1])  // Convert first regex capture group to int
	// 		episode, _ := strconv.Atoi(values[0][2]) // Convert second regex capture group to int
	// 		meta.Season = season
	// 		meta.Episode = episode
	// 		break
	// 	} else {
	// 		// Add the word to our show tags
	// 		meta.ShowTags = append(meta.ShowTags, word)
	// 	}
	// }
	// for i := range meta.ShowTags {
	// 	meta.ShowTags[i] = strings.Title(meta.ShowTags[i])
	// }
	return meta
}

func CleanTitle(title, ignoreChars string) string {
	titleRunes := []rune(title)
	ignoreCharsRunes := []rune(ignoreChars)
	for i := range titleRunes {
		for j := range ignoreCharsRunes {
			if titleRunes[i] == ignoreCharsRunes[j] {
				titleRunes[i] = ' '
				break
			}
		}
	}

	split := strings.Split(string(titleRunes), " ")
	pos := 0
	for i := range split {
		if split[i] != "" { // Ignore empty segments between pairs of ignoreChars
			split[pos] = strings.Title(split[i]) // Make each word title case
			pos++
		}
	}
	split = split[:pos]

	return strings.Join(split, " ")
}

func isSeparator(r rune) bool {
	return r == '.' || r == ' '
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("'%s' is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
