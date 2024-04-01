package main

import (
	"embed"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Mime types for different files
var MimeTypes = map[string]string{
	".css":  "text/css",
	".js":   "application/javascript",
	".icon": "image-x-icon",
	".svg":  "image/svg+xml",
}

// Error struct
type Error struct {
	Message string `json:"error"`
}

type Info struct {
	Connection []Connection `json:"connections"`
}

//go:embed static
var staticFolder embed.FS

// NewError creates new Error struct from go's error
func NewError(err error) Error {
	return Error{err.Error()}
}

func assetContentType(name string) string {
	mime := MimeTypes[filepath.Ext(name)]

	if mime != "" {
		return mime
	}

	return "text/plain"
}

// APIHome load home page
func APIHome(c *gin.Context) {
	data, err := staticFolder.ReadFile("static/index.html")
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

// APIConnect will connect to our mysql database
func APIConnect(c *gin.Context) {
	url := c.Request.FormValue("url")

	if url == "" {
		c.JSON(http.StatusBadRequest, Error{"Url parameter is required"})
		return
	}

	clientKey, err := NewClientFromURL(url)
	if err != nil {
		c.JSON(http.StatusBadRequest, Error{err.Error()})
		return
	}

	client := dbClientMap[clientKey]

	err = client.Test()
	if err != nil {
		c.JSON(http.StatusBadRequest, Error{err.Error()})
		return
	}

	user, host, database, port := getConnParametersFromString(url)
	dbConn := Connection{
		Host:     host,
		Port:     port,
		Username: user,
		Database: database,
		ConnID:   clientKey,
	}

	dbConnArr = append(dbConnArr, dbConn)

	info, err := client.Info()

	formatedRes := info.Format()[0]

	formatedRes["connId"] = clientKey

	c.JSON(http.StatusOK, formatedRes)
}

func APIClose(c *gin.Context) {
	// Read client id from the headers
	dbClientKey := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[dbClientKey]

	err := dbClient.Close()
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
	}

	// Remove from
	delete(dbClientMap, dbClientKey)
	for index, element := range dbConnArr {
		thisConnId := element.ConnID

		if thisConnId == dbClientKey {
			dbConnArr = append(dbConnArr[:index], dbConnArr[index+1:]...)
			break
		}
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

// APIGetDatabases will get you all databases in system
func APIGetDatabases(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	names, err := dbClient.Databases()
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, names)
}

// APIGetDatabaseTables will give the tables of a database
func APIGetDatabaseTables(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.DatabaseTables(c.Params.ByName("database"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIGetDatabaseViews will give the views of a database
func APIGetDatabaseViews(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.DatabaseViews(c.Params.ByName("database"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIGetDatabaseProcedures will give the stored procedures of a database
func APIGetDatabaseProcedures(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.DatabaseProcedures(c.Params.ByName("database"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIGetDatabaseFunctions will give the functions of a database
func APIGetDatabaseFunctions(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.DatabaseFunctions(c.Params.ByName("database"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APISetDefaultDatabase will set the database as default db for connection
func APISetDefaultDatabase(c *gin.Context) {
	dbName := c.Params.ByName("database")
	query := fmt.Sprintf("use %s;", dbName)

	APIHandleQuery(query, c)
}

// APIRunQuery will run the user's sql query
func APIRunQueryGet(c *gin.Context) {
	query := strings.TrimSpace(c.Request.FormValue("query"))

	if query == "" {
		c.JSON(http.StatusBadRequest, errors.New("Query parameter is missing"))
		return
	}

	APIHandleQuery(query, c)
}

func APIRunQuery(c *gin.Context) {
	query := strings.TrimSpace(c.Request.FormValue("query"))

	if query == "" {
		c.JSON(http.StatusBadRequest, errors.New("Query parameter is missing"))
		return
	}

	APIHandleQuery(query, c)
}

// APIExplainQuery will run explain on the sql query and return the output
func APIExplainQuery(c *gin.Context) {
	query := strings.TrimSpace(c.Request.FormValue("query"))

	if query == "" {
		c.JSON(http.StatusBadRequest, errors.New("Query parameter is missing"))
		return
	}

	APIHandleQuery(fmt.Sprintf("EXPLAIN %s", query), c)
}

func APIGetColumnOfTable(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.TableColumns(c.Params.ByName("database"), c.Params.ByName("table"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIGetTableInfo returns info about table like row_count, data size etc.
func APIGetTableInfo(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.TableInfo(c.Params.ByName("table"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res.Format()[0])
}

// APIHistory will return query history of current dbClient
func APIHistory(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	c.JSON(http.StatusOK, dbClient.history)
}

// APIInfo returns information about the current db connecction
func APIInfo(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	if dbClient == nil {
		// Also send the available connections list

		formatedRes := &Info{
			Connection: dbConnArr,
		}

		c.JSON(http.StatusBadRequest, formatedRes)
		return
	}

	res, err := dbClient.Info()
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	formatedRes := res.Format()[0]

	formatedRes["host"] = dbClient.host
	formatedRes["user"] = dbClient.user

	c.JSON(http.StatusOK, formatedRes)
}

// APITableIndexes returns the indexs of a table
func APITableIndexes(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.TableIndexes(c.Params.ByName("table"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIProcedureParameters returns the parameters of a procedure
func APIProcedureParameters(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.ProcedureParameters(c.Params.ByName("procedure"), c.Request.FormValue("database"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIGetCollationCharSet returns the character sets and collation available in
// database
func APIGetCollationCharSet(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.DatabaseCollationCharSet()
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIAlterDatabase alter database to change charset & collation
func APIAlterDatabase(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.AlterDatabase(c.Params.ByName("database"),
		c.Request.FormValue("charset"), c.Request.FormValue("collation"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusCreated, res)
}

// APIDropDatabase drops the given database from the system
func APIDropDatabase(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	_, err := dbClient.DropDatabase(c.Params.ByName("database"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

// APIDropTable will drop the table from this database
func APIDropTable(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	_, err := dbClient.DropTable(c.Params.ByName("database"), c.Params.ByName("table"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

// APITruncateTable truncates the table
func APITruncateTable(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	_, err := dbClient.TruncateTable(c.Params.ByName("database"), c.Params.ByName("table"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

// APIProcedureDefinition get definition of a procedure
func APIProcedureDefinition(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.ProcedureDefinition("procedure", c.Params.ByName("database"), c.Params.ByName("procedure"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIFunctionDefinition get definition of a function
func APIFunctionDefinition(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.ProcedureDefinition("function", c.Params.ByName("database"), c.Params.ByName("function"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APICreateProcedure creates/edits a stored procedure
func APICreateProcedure(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	dbName := c.Params.ByName("database")
	procName := c.Params.ByName("procedure")
	procDef := c.Request.FormValue("definition")

	_, err := dbClient.ProcedureCreate("PROCEDURE", dbName, procName, procDef)
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
}

// APICreateFunction creates/edits a function
func APICreateFunction(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	dbName := c.Params.ByName("database")
	procName := c.Params.ByName("function")
	procDef := c.Request.FormValue("definition")

	_, err := dbClient.ProcedureCreate("FUNCTION", dbName, procName, procDef)
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
}

// APIDropProcedure drops the procedure
func APIDropProcedure(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	_, err := dbClient.DropProcedure("PROCEDURE", c.Params.ByName("database"), c.Params.ByName("procedure"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

// APIViewDefinition gets the definition of a view
func APIViewDefinition(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.ViewDefinition(c.Params.ByName("database"), c.Params.ByName("view"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

func apiSearch(c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")
	dbClient := dbClientMap[yoConnID]

	res, err := dbClient.Search(c.Params.ByName("query"))
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, res)
}

// APIHandleQuery handles thq query and return the resultset as JSON
func APIHandleQuery(query string, c *gin.Context) {
	// Read client id from the headers
	yoConnID := c.Request.Header.Get("X-CONN-ID")

	// If id missing from header, check in query string
	if yoConnID == "" {
		yoConnID = c.Request.FormValue("conn_id")
	}

	if yoConnID == "" {
		c.JSON(http.StatusBadRequest, Error{"Invalid connection"})
		return
	}

	dbClient := dbClientMap[yoConnID]

	// 31 Aug
	// Make it mandatory to have WHERE for UPDATE & DELETE
	// TODO: Make this enforcing a setting

	if strings.Contains(strings.ToUpper(query), "UPDATE") ||
		strings.Contains(strings.ToUpper(query), "DELETE") {
		if !strings.Contains(strings.ToUpper(query), "WHERE") {
			c.JSON(http.StatusBadRequest, Error{"WHERE statement is mandatory with UPDATE & DELETE statements"})
			return
		}
	}

	result, err := dbClient.Query(query)
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	q := c.Request.URL.Query()

	if len(q["format"]) > 0 {
		if q["format"][0] == "csv" {
			c.Data(http.StatusOK, "text/csv", result.CSV())
			return
		}
	}

	c.JSON(http.StatusOK, result)
}

func APIGetBookmarks(c *gin.Context) {
	bookmarks, err := readBookmarks(getBookmarkPath())
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.JSON(http.StatusOK, bookmarks)
}

func APISaveBookmark(c *gin.Context) {
	bookName := c.Params.ByName("name")

	conHost := c.Request.FormValue("host")
	strConPort := c.Request.FormValue("port")
	intConPort, err := strconv.Atoi(strConPort)
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}
	conUser := c.Request.FormValue("user")
	conDatabase := c.Request.FormValue("database")

	objBookmark := Bookmark{
		Name: bookName,
		Connection: Connection{
			Host:     conHost,
			Port:     intConPort,
			Username: conUser,
			Database: conDatabase,
		},
	}

	i, err := saveBookmark(objBookmark, getBookmarkPath())

	if i == -1 {
		c.JSON(http.StatusBadRequest, NewError(errors.New("A connection with this name already exists")))
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

func APIDeleteBookmark(c *gin.Context) {
	bookName := c.Params.ByName("name")

	err := deleteBookmark(bookName, getBookmarkPath())
	if err != nil {
		c.JSON(http.StatusBadRequest, NewError(err))
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

// APIServeAsset serves the static assets
func APIServeAsset(c *gin.Context) {
	file := fmt.Sprintf(
		"static%s",
		c.Params.ByName("filepath"),
	)

	data, err := staticFolder.ReadFile(file)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	if len(data) == 0 {
		c.String(http.StatusNotFound, "Asset is empty")
		return
	}

	c.Data(http.StatusOK, assetContentType(file), data)
}

func getUpdate(c *gin.Context) {
	c.Writer.WriteHeader(http.StatusNoContent)
	// update := checkForUpdate(VERSION)

	// if update == nil {
	// 	return
	// }

	// c.JSON(http.StatusOK, update)
}
