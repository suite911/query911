package query

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

var Logger *log.Logger
var Verbose bool

type Query struct {
	Result     sql.Result
	Error      error
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
		query.logMethod("DB.Begin", err)
	}
}

// Close query.Rows if open; a no-op otherwise
func (query *Query) Close() {
	if query.isOpen {
		query.isOpen = false
		query.logMethod("Rows.Close", query.Rows.Close())
	}
}

// Try to commit pending changes; roll back if anything goes wrong
func (query *Query) CommitOrRollback() (ok bool) {
	if query.OK() && query.okToCommit {
		err := query.Tx.Commit()
		query.logMethod("Tx.Commit", err)
		return err == nil
	} else {
		query.logMethod("Tx.Rollback", query.Tx.Rollback())
		return false
	}
}

// Clear the error stack and accumulated log text
func (query *Query) ErrorClear() {
	query.Error = nil
	query.LogText = query.LogText[:0]
}

// Push an error onto the error stack, assuming it was caused by previous errors, if present
func (query *Query) ErrorPush(err error, msg ...interface{}) {
	if err == nil {
		err = errors.New("(nil error)")
	}
	msgs := strings.Replace(strings.Replace(fmt.Sprintf("{%q: %q}",
		fmt.Sprint(msg), err), `\`, `\\`, -1), "\n", `\n`, -1)

	if cur := query.Error; cur != nil {
		query.Error = errors.Wrap(cur, msgs)
	} else {
		query.Error = errors.Wrap(err, msgs)
	}
}

// Return the error stack in a format more suitable for printing
func (query *Query) ErrorStack() (string, errors.StackTrace) {
	var earliestStackTrace errors.StackTrace
	var errors string
	e := query.Error
	for {
		if len(errors) > 0 {
			errors += "\n"
		}
		errors += e.Error()
		if st, ok := e.(errors.StackTracer); ok {
			earliestStackTrace = st.StackTrace()
		}
		if c, ok := e.(errors.Causer); ok {
			if e = c.Cause(); e != nil {
				continue
			}
		}
		break
	}
	return errors, earliestStackTrace
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
		query.logMethod("Stmt.Exec", err)
	}
}

// Get the error message in a format suitable for use with the Discord API
// It probably looks good on the console too
func (query *Query) GetErrorDiscord() error {
	if query.OK() {
		return nil
	}
	text := "SQL `QueryError`:\n```sql\n"
	text += query.SQL
	text += "\u200b\n```"
	errmsg := query.LastError().Error()
	if len(errmsg) > 0 {
		text += "\nError: \""
		text += errmsg
		text += "\""
	}
	if len(query.LogText) > 0 {
		text += "\nLog:\n```go"
		text += query.LogText
		text += "\u200b\n```"
	}
	return errors.New(text)
}

// Get the last error to occur
func (query *Query) LastError() (err error) {
	count := len(query.Errors)
	if count > 0 {
		err = query.Errors[count-1]
		if err == nil {
			panic(nil)
		}
		return
	}
	return nil
}

// Log the accumulated log text to the logger now if there was an error and return the last error
// Example: if nil != query.LogErrors() { return }
func (query *Query) LogErrors() (err error) {
	err = query.LastError()
	if err != nil {
		query.LogNow()
	}
	return err
}

// Log the accumulated log text to the logger now
func (query *Query) LogNow() {
	query.logPrintln(query.LogText)
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
	return len(query.Errors) < 1
}

// Log the accumulated log text to the logger now if there was an error and panic if so
// Example: query.PanicErrors()
func (query *Query) PanicErrors() {
	err := query.LastError()
	if err != nil {
		query.LogNow()
		panic(err)
	}
}

// For use with query.ExecPrepared or query.QueryPrepared
func (query *Query) Prepare() {
	if query.OK() {
		query.logSQL()
		var err error
		if query.Tx != nil {
			query.Stmt, err = query.Tx.Prepare(query.SQL)
			query.logMethod("Tx.Prepare", err)
		} else {
			query.Stmt, err = query.DB.Prepare(query.SQL)
			query.logMethod("DB.Prepare", err)
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
		query.logMethod("Stmt.Query", err)
		query.isOpen = query.OK()
	}
}

// Calls query.Rows.Scan and then query.Rows.Close
func (query *Query) ScanClose(args ...interface{}) {
	if query.OK() {
		query.logMethod("Rows.Scan", query.Rows.Scan(args...))
		query.Close()
	}
}

// Like query.ScanKeepOpen, but does not call query.Rows.Close afterward
func (query *Query) ScanKeepOpen(args ...interface{}) {
	if query.OK() {
		query.logMethod("Rows.Scan", query.Rows.Scan(args...))
	}
}

func (query *Query) logMethod(method string, err error) {
	if err != nil || query.verbose() {
		wasOK := query.OK()
		if err != nil {
			query.Errors = append(query.Errors, err)
		}
		if len(query.LogText) > 0 {
			query.LogText += "\n"
		}
		query.LogText += "query."
		query.LogText += method
		query.LogText += "() // "
		if err == nil {
			query.LogText += "ok"
		} else {
			query.LogText += "error: \""
			query.LogText += err.Error()
			query.LogText += "\""
		}
		if wasOK {
			query.LogText += " // stored"
		}
	}
}

func (query *Query) logPrintln(args ...interface{}) {
	switch {
	case query.Logger != nil:
		query.Logger.Println(args...)
	case Logger != nil:
		Logger.Println(args...)
	default:
		log.Println(args...)
	}
}

func (query *Query) logSQL() {
	query.LogText += "\u200b\n```\nSQL set to:\n```sql\n"
	query.LogText += query.SQL
	query.LogText += "\u200b\n```\n```go"
}

func (query *Query) verbose() bool {
	return query.Verbose || Verbose
}
