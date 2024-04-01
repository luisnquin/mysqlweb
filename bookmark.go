package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

// Connection is a single saved connection-string object
type Connection struct {
	Host     string
	Port     int
	Username string
	Database string
	ConnID   string
}

type Bookmark struct {
	Name       string     `json:"name"`
	Connection Connection `json:"conn_info"`
}

type Bookmarks struct {
	Bookmarks []Bookmark `json:"bookmarks"`
}

func fileBaseName(path string) string {
	filename := filepath.Base(path)
	return strings.Replace(filename, filepath.Ext(path), "", 1)
}

func getBookmarkPath() string {
	path, _ := homedir.Dir()
	bookmarkPath := fmt.Sprintf("%s/.mysqlweb/bookmark", path)

	// Check if this path exists
	isExists, _ := ExistsFileFolder(bookmarkPath)

	// If no, then create directory
	if !isExists {
		os.MkdirAll(bookmarkPath, 0o777)
	}

	return bookmarkPath
}

func readBookmarks(path string) (Bookmarks, error) {
	results := Bookmarks{
		Bookmarks: []Bookmark{},
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return results, err
	}

	for _, file := range files {
		// We need .json files only
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		fullpath := filepath.FromSlash(path + "/" + file.Name())
		conName := fileBaseName(file.Name())

		data, err := ioutil.ReadFile(fullpath)
		if err != nil {
			return results, err
		}

		thisConn := Connection{}

		json.Unmarshal(data, &thisConn)

		thisBookmark := Bookmark{
			Name:       conName,
			Connection: thisConn,
		}

		results.Bookmarks = append(results.Bookmarks, thisBookmark)

	}

	return results, nil
}

func saveBookmark(objBookmark Bookmark, path string) (int, error) {
	fileName := objBookmark.Name + ".json"

	data, jerr := json.MarshalIndent(objBookmark.Connection, "", "  ")
	if jerr != nil {
		return 0, jerr
	}

	fullFilePath := filepath.FromSlash(path + "/" + fileName)

	// equivalent to Python's `if not os.path.exists(filename)`
	if _, err := os.Stat(fullFilePath); err == nil {
		// fmt.Printf("file exists; processing...")
		return -1, nil
	}

	filErr := ioutil.WriteFile(fullFilePath, data, 0o644)
	if filErr != nil {
		return 0, filErr
	}

	return 1, nil
}

func deleteBookmark(bookmarkName string, path string) error {
	fileName := bookmarkName + ".json"

	fullFilePath := filepath.FromSlash(path + "/" + fileName)

	err := os.Remove(fullFilePath)
	if err != nil {
		return err
	}

	return nil
}
