package query

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

var Logger *log.Logger
var Verbose bool

type Query struct {
	Result     sql.Result
	Error      Error
	SQL        string
	LogText    string
	Logger     *log.Logger
	DB         *sql.DB
	Tx         *sql.Tx
	Stmt       *sql.Stmt
	Rows       *sql.Rows
	Verbose    bool
	isOpen     bool
	okToCommit bool
}

// Constructor
func New(db *sql.DB) *Query {
	return new(Query).Init(db)
}

// Initialize your own Query
func (q *Query) Init(db *sql.DB) *Query {
	q.DB = db
	return q
}

// Begin a transaction
func (query *Query) Begin() {
	if query.OK() {
		var err error
		query.Tx, err = query.DB.Begin()
		query.LogMethodCall("DB.Begin", err)
	}
}

// Close query.Rows if open; a no-op otherwise
func (query *Query) Close() {
	if query.isOpen {
		query.isOpen = false
		query.LogMethodCall("Rows.Close", query.Rows.Close())
	}
}

// Try to commit pending changes; roll back if anything goes wrong
func (query *Query) CommitOrRollback() (ok bool) {
	if query.OK() && query.okToCommit {
		err := query.Tx.Commit()
		query.LogMethodCall("Tx.Commit", err)
		return err == nil
	} else {
		query.LogMethodCall("Tx.Rollback", query.Tx.Rollback())
		return false
	}
}

// Essentially calls query.Prepare and query.ExecPrepared
func (query *Query) Exec(args ...interface{}) {
	if query.OK() {
		query.Prepare()
		query.ExecPrepared(args...)
	}
}

// Call query.Prepare first
func (query *Query) ExecPrepared(args ...interface{}) {
	if query.OK() {
		var err error
		query.Result, err = query.Stmt.Exec(args...)
		query.LogMethodCall("Stmt.Exec", err)
	}
}

// Exposed publically but generally only used internally to log internal method calls.
func (query *Query) LogMethodCall(method string, err error) {
	if err != nil || query.verbose() {
		wasOK := query.OK()
		if err != nil {
			query.Error.Push(method + ":", err)
		}
		comment := " // ok"
		if err != nil {
			comment = fmt.Sprint(" // error: %q", err)
		}
		if wasOK {
			comment += " // stored"
		}
		query.Error.Log("go", method + comment)
	}
}

// Log any errors to the logger now and return `Error` as an `error`
func (query *Query) LogNow() error {
	err := query.Error
	if err != nil {
		query.loggerPrintln(err.ErrorText())
	}
	return err
}

// Like `LogNow`, but open any errors in a browser window for easy debugging
func (query *Query) LogNowBrowser() error {
	err := query.Error
	if err != nil {
		query.loggerPrintln(err.ErrorText())
		err.ErrorBrowser()
	}
	return err
}

// Exposed publically but generally only used internally to log SQL code.
func (query *Query) LogSQL() {
	query.Error.Log("sql", "SQL code:", query.SQL)
}

// Calls query.Rows.Next and returns if there are any more rows
func (query *Query) NextKeepOpen() (hasNext bool) {
	return query.OK() && query.Rows.Next()
}

// Like query.NextKeepOpen, but call query.Rows.Close if there are no more rows
func (query *Query) NextOrClose() (hasNext bool) {
	hasNext = query.OK() && query.Rows.Next()
	if !hasNext {
		query.Close()
	}
	return
}

// Whether (false) or not (true) there was an error
func (query *Query) OK() bool {
	return query.Error == nil
}

// For use with query.ExecPrepared or query.QueryPrepared
func (query *Query) Prepare() {
	if query.OK() {
		query.LogSQL()
		var err error
		if query.Tx != nil {
			query.Stmt, err = query.Tx.Prepare(query.SQL)
			query.LogMethodCall("Tx.Prepare", err)
		} else {
			query.Stmt, err = query.DB.Prepare(query.SQL)
			query.LogMethodCall("DB.Prepare", err)
		}
	}
}

// Essentially calls query.Prepare and query.QueryPrepared
func (query *Query) Query(args ...interface{}) {
	if query.OK() {
		query.Prepare()
		query.QueryPrepared(args...)
	}
}

// Call query.Prepare first
func (query *Query) QueryPrepared(args ...interface{}) {
	if query.OK() {
		var err error
		query.Rows, err = query.Stmt.Query(args...)
		query.LogMethodCall("Stmt.Query", err)
		query.isOpen = query.OK()
	}
}

// Calls query.Rows.Scan and then query.Rows.Close
func (query *Query) ScanClose(args ...interface{}) {
	if query.OK() {
		query.LogMethodCall("Rows.Scan", query.Rows.Scan(args...))
		query.Close()
	}
}

// Like query.ScanKeepOpen, but does not call query.Rows.Close afterward
func (query *Query) ScanKeepOpen(args ...interface{}) {
	if query.OK() {
		query.LogMethodCall("Rows.Scan", query.Rows.Scan(args...))
	}
}

func (query *Query) loggerPrintln(args ...interface{}) {
	switch {
	case query.Logger != nil:
		query.Logger.Println(args...)
	case Logger != nil:
		Logger.Println(args...)
	default:
		log.Println(args...)
	}
}

func (query *Query) verbose() bool {
	return query.Verbose || Verbose
}
