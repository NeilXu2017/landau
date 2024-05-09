package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/util"

	//导入mysql驱动
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

type (
	//Database MySQL数据库连接串信息
	Database struct {
		loc                      string
		timeZone                 string
		dbUser                   string
		dbPassword               string
		host                     string
		port                     int
		schemaName               string
		dbConnection             string
		writeLog                 bool
		logger                   string
		maxOpenConnections       int
		maxIdleConnections       int
		maxConnectionLifeTime    int
		keepAllConnection        bool
		lastSetKeptAllConnection bool
		driverName               string
		driverExtendDSNProperty  map[string]interface{}
		customLogSQL             func(string) string
		txOptions                *sql.TxOptions
	}
	_TxWrap struct {
		start           time.Time
		tx              *sql.Tx
		execSQLSequence int
		db              *Database
	}
	//TxExecute 业务提供给事务管理的回调函数，业务使用 _TxWrap 对象执行SQL,根据回调函数最后返回的结果决定提交或回滚事务
	TxExecute func(c Execute) error
	//Execute 事务对象执行SQL
	Execute interface {
		Exec(strSQL string, args ...interface{}) (int, int, error)
		Get(dbModel interface{}, strSQL string, args ...interface{}) (int, error)
		Gets(dbModel interface{}, strSQL string, args ...interface{}) (int, error)
		ScanGet(strSQL string, dest ...interface{}) (int, error) //临时提供的方法
	}
	// DatabaseOptionFunc 参数设置
	DatabaseOptionFunc func(*Database) error
)

const (
	dbConnectionFmtNoTimeZone      = "%s:%s@tcp(%s:%d)/%s"
	dbConnectionFmt                = "%s:%s@tcp(%s:%d)/%s?loc=%s&time_zone=%s"
	defaultDbLogger                = "main"
	defaultMaxOpenConnections      = 50
	defaultMaxIdleConnections      = 50
	defaultMaxConnectionLifeTime   = 600
	defaultKeepAllConnection       = false
	mysqlEscapeBackslash           = `\`
	mysqlEscapeBackslashReplaced   = `\\`
	mysqlEscapeSingleQuote         = `'`
	mysqlEscapeSingleQuoteReplaced = `\'`
	defaultMySQLDriverName         = "mysql"
	//ClickHouseDriver driver Name
	ClickHouseDriver = "clickhouse"
)

var (
	defaultDatabase  *Database
	dbConnectionPool sync.Map //key: dbConnection value: *sqlx.DB
)

// SetDatabaseHost 设置 Database Server 地址
func SetDatabaseHost(host string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.host = host
		return nil
	}
}

// SetDatabasePort 设置 Database Server 端口
func SetDatabasePort(port int) DatabaseOptionFunc {
	return func(c *Database) error {
		c.port = port
		return nil
	}
}

// SetDatabaseUser 设置 Database 用户
func SetDatabaseUser(dbUser string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.dbUser = dbUser
		return nil
	}
}

// SetDatabasePassword 设置 Database 用户密码
func SetDatabasePassword(dbPassword string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.dbPassword = dbPassword
		return nil
	}
}

// SetDatabaseSchema 设置 Database 数据库名
func SetDatabaseSchema(schemaName string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.schemaName = schemaName
		return nil
	}
}

// SetDatabaseLogger 设置 Database 日志 logger 名称
func SetDatabaseLogger(loggerName string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.logger = loggerName
		return nil
	}
}

// SetDatabaseTraceLog 设置 Database 是否记录SQL执行日志
func SetDatabaseTraceLog(writeLog bool) DatabaseOptionFunc {
	return func(c *Database) error {
		c.writeLog = writeLog
		return nil
	}
}

// SetDatabaseMaxOpenConnections 设置最大链接个数
func SetDatabaseMaxOpenConnections(maxOpenConnections int) DatabaseOptionFunc {
	return func(c *Database) error {
		c.maxOpenConnections = maxOpenConnections
		return nil
	}
}

// SetDatabaseMaxIdleConnections 设置最大空闲链接个数
func SetDatabaseMaxIdleConnections(maxIdleConnections int) DatabaseOptionFunc {
	return func(c *Database) error {
		c.maxIdleConnections = maxIdleConnections
		return nil
	}
}

// SetDatabaseMaxConnectionLifeTime 设置最大链接时间
func SetDatabaseMaxConnectionLifeTime(maxConnectionLifeTime int) DatabaseOptionFunc {
	return func(c *Database) error {
		c.maxConnectionLifeTime = maxConnectionLifeTime
		return nil
	}
}

// SetDatabaseLoc 设置地区
func SetDatabaseLoc(loc string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.loc = loc
		return nil
	}
}

// SetDatabaseTimZone 设置时区
func SetDatabaseTimZone(timeZone string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.timeZone = timeZone
		return nil
	}
}

// SetDatabaseDriverName  设置driverName
func SetDatabaseDriverName(driverName string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.driverName = driverName
		return nil
	}
}

// SetDatabaseExtendDSNProperty 设置扩展属性
func SetDatabaseExtendDSNProperty(dsnProperty map[string]interface{}) DatabaseOptionFunc {
	return func(c *Database) error {
		c.driverExtendDSNProperty = dsnProperty
		return nil
	}
}

func SetDatabaseCustomLogSQL(f func(string) string) DatabaseOptionFunc {
	return func(c *Database) error {
		c.customLogSQL = f
		return nil
	}
}

func SetDatabaseTxOptions(txOptions *sql.TxOptions) DatabaseOptionFunc {
	return func(c *Database) error {
		c.txOptions = txOptions
		return nil
	}
}

// GetDB 返回sqlx.DB 对象
func (c *Database) GetDB() (*sqlx.DB, error) {
	if conn, ok := dbConnectionPool.Load(c.dbConnection); ok {
		dbConn := conn.(*sqlx.DB)
		if c.keepAllConnection != c.lastSetKeptAllConnection {
			maxIdleConnNum := c.maxIdleConnections
			if c.keepAllConnection {
				maxIdleConnNum = c.maxOpenConnections
			}
			dbConn.SetMaxIdleConns(maxIdleConnNum)
			c.lastSetKeptAllConnection = c.keepAllConnection
		}
		return dbConn, nil
	}
	dbConn, err := sqlx.Open(c.driverName, c.dbConnection)
	if err == nil {
		dbConn.SetMaxOpenConns(c.maxOpenConnections)
		maxIdleConnNum := c.maxIdleConnections
		if c.keepAllConnection {
			maxIdleConnNum = c.maxOpenConnections
		}
		dbConn.SetMaxIdleConns(maxIdleConnNum)
		dbConn.SetConnMaxLifetime(time.Duration(c.maxConnectionLifeTime) * time.Second)
		c.lastSetKeptAllConnection = c.keepAllConnection
		dbConnectionPool.Store(c.dbConnection, dbConn)
	}
	return dbConn, err
}

// SetKeepAllIdleConn 设置是否保持空闲DB链接
func (c *Database) SetKeepAllIdleConn(keepAllConn bool) {
	c.keepAllConnection = keepAllConn
}

// NewDatabase 构建Database对象
func NewDatabase(options ...DatabaseOptionFunc) *Database {
	db := &Database{
		loc:                   "",
		timeZone:              "",
		maxOpenConnections:    defaultMaxOpenConnections,
		maxIdleConnections:    defaultMaxIdleConnections,
		maxConnectionLifeTime: defaultMaxConnectionLifeTime,
		keepAllConnection:     defaultKeepAllConnection,
		writeLog:              true,
		logger:                defaultDbLogger,
		driverName:            defaultMySQLDriverName,
		txOptions: &sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
			ReadOnly:  false,
		},
	}
	for _, option := range options {
		_ = option(db)
	}
	strDBConnection := ""
	switch db.driverName {
	case ClickHouseDriver:
		strDBConnection = fmt.Sprintf("tcp://%s:%d?username=%s&password=%s&database=%s", util.IPConvert(db.host, util.IPV6Bracket), db.port, db.dbUser, db.dbPassword, db.schemaName)
	default:
		if db.loc == "" {
			strDBConnection = fmt.Sprintf(dbConnectionFmtNoTimeZone, db.dbUser, db.dbPassword, util.IPConvert(db.host, util.IPV6Bracket), db.port, db.schemaName)
		} else {
			strDBConnection = fmt.Sprintf(dbConnectionFmt, db.dbUser, db.dbPassword, util.IPConvert(db.host, util.IPV6Bracket), db.port, db.schemaName, db.loc, url.QueryEscape(db.timeZone))
		}
	}
	for k, v := range db.driverExtendDSNProperty {
		if strings.Contains(strDBConnection, "?") {
			strDBConnection = fmt.Sprintf("%s&%s=%v", strDBConnection, k, v)
		} else {
			strDBConnection = fmt.Sprintf("%s?%s=%v", strDBConnection, k, v)
		}
	}
	db.dbConnection = strDBConnection
	return db
}

// NewDatabase2 构建Database对象
func NewDatabase2(host string, port int, user, password, schema string) *Database {
	return NewDatabase(SetDatabaseHost(host), SetDatabasePort(port), SetDatabaseUser(user), SetDatabasePassword(password), SetDatabaseSchema(schema))
}

// NewDatabase3 构建Database对象
func NewDatabase3(host string, port int, user, password, schema string, writeLog bool) *Database {
	return NewDatabase(SetDatabaseHost(host), SetDatabasePort(port), SetDatabaseUser(user), SetDatabasePassword(password), SetDatabaseSchema(schema), SetDatabaseTraceLog(writeLog))
}

// NewDefaultDatabase 构建缺省Database对象
func NewDefaultDatabase(host string, port int, user, password, schema string) {
	defaultDatabase = NewDatabase2(host, port, user, password, schema)
}

func NewDefaultDatabase2(host string, port int, user, password, schema string, customLogSQL func(string) string) {
	defaultDatabase = NewDatabase2(host, port, user, password, schema)
	defaultDatabase.customLogSQL = customLogSQL
}

func getArrayLen(v reflect.Value) int {
	switch v.Kind() {
	case reflect.Array:
		return v.Len()
	case reflect.Slice:
		return v.Len()
	case reflect.Ptr:
		return getArrayLen(v.Elem())
	default:
		return 0
	}
}

func _getArgsLog(logArgs func(args ...interface{}) string, args ...interface{}) interface{} {
	if logArgs == nil {
		return args
	}
	return logArgs(args...)
}

func _getDBResultLog(logResult func(r interface{}) string, result interface{}) interface{} {
	if logResult == nil {
		return result
	}
	return logResult(result)
}

func (c *Database) query(dbModel interface{}, strSQL string, isGetOne bool, logArgs func(args ...interface{}) string, logResult func(r interface{}) string, args ...interface{}) (int, error) {
	start := time.Now()
	db, err := c.GetDB()
	if err != nil {
		log.Error2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\tError:[%v]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), err)
		return 0, err
	}
	if isGetOne {
		err = db.Get(dbModel, strSQL, args...)
	} else {
		err = db.Select(dbModel, strSQL, args...)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\t[row count=0]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...))
			return 0, nil
		}
		log.Error2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\tError:[%v]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), err)
		return 0, err
	}
	rowCounts := 1
	if !isGetOne {
		rowCounts = getArrayLen(reflect.ValueOf(dbModel))
	}
	if c.writeLog && log.IsEnableCategoryInfoLog("SQL") {
		if isGetOne {
			log.Info2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\t[%v]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), _getDBResultLog(logResult, dbModel))
		} else {
			log.Info2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\t[row count=%d]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), rowCounts)
		}
	}
	return rowCounts, nil
}

// GetDbConn 返回DB连接字符串
func (c *Database) GetDbConn() string {
	return c.dbConnection
}

// Get 查询单个记录
func (c *Database) Get(dbModel interface{}, strSQL string, args ...interface{}) (int, error) {
	return c.query(dbModel, strSQL, true, nil, nil, args...)
}
func (c *Database) Get2(dbModel interface{}, strSQL string, logArgs func(args ...interface{}) string, logResult func(r interface{}) string, args ...interface{}) (int, error) {
	return c.query(dbModel, strSQL, true, logArgs, logResult, args...)
}

// Gets  查询多条记录
func (c *Database) Gets(dbModel interface{}, strSQL string, args ...interface{}) (int, error) {
	return c.query(dbModel, strSQL, false, nil, nil, args...)
}

func (c *Database) getLogSQL(sql string) string {
	if c.customLogSQL == nil {
		return sql
	}
	return c.customLogSQL(sql)
}

// Exec 执行SQL语句，返回影响的行数，如果数据库驱动不支持，返回-1
func (c *Database) Exec(strSQL string, args ...interface{}) (int, error) {
	return c.Exec2(strSQL, nil, args...)
}

func (c *Database) Exec2(strSQL string, logArgs func(args ...interface{}) string, args ...interface{}) (int, error) {
	start := time.Now()
	db, err := c.GetDB()
	if err != nil {
		log.Error2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\tError:[%v]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), err)
		return 0, err
	}
	result, execError := db.Exec(strSQL, args...)
	if execError != nil {
		log.Error2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\tError:[%v]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), execError)
		return 0, execError
	}
	rowsCount, affectedError := result.RowsAffected()
	if affectedError != nil {
		rowsCount = -1
	}
	if c.writeLog && log.IsEnableCategoryInfoLog("SQL") {
		log.Info2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\t[Affected Row Counts=%d]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), int(rowsCount))
	}
	return int(rowsCount), nil
}

// Insert 执行插入SQL语句，如果数据库驱动支持，且表有自增类型的主键，返回插入记录的主键，否则返回-1
func (c *Database) Insert(strSQL string, args ...interface{}) (int, error) {
	return c.Insert2(strSQL, nil, args...)
}

func (c *Database) Insert2(strSQL string, logArgs func(args ...interface{}) string, args ...interface{}) (int, error) {
	start := time.Now()
	db, err := c.GetDB()
	if err != nil {
		log.Error2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\tError:[%v]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), err)
		return 0, err
	}
	result, execError := db.Exec(strSQL, args...)
	if execError != nil {
		log.Error2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\tError:[%v]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), execError)
		return 0, execError
	}
	lastInsertID, lastInsertErr := result.LastInsertId()
	if lastInsertErr != nil {
		lastInsertID = -1
	}
	if c.writeLog && log.IsEnableCategoryInfoLog("SQL") {
		log.Info2(c.logger, "[SQL] [%s]\t[%s]\tArgs [%v]\t[Last Inserted Row Id=%d]", time.Since(start), c.getLogSQL(strSQL), _getArgsLog(logArgs, args...), int(lastInsertID))
	}
	return int(lastInsertID), nil
}

// ExecTx 开始一个事务执行一系列SQL
func (c *Database) ExecTx(bizFunc TxExecute) error {
	start := time.Now()
	db, err := c.GetDB()
	if err != nil {
		log.Error2(c.logger, "[SQL ExecTx] [%s]\t Error:[%v]", time.Since(start), err)
		return err
	}
	tx, err := db.BeginTx(context.Background(), c.txOptions)
	if err != nil {
		log.Error2(c.logger, "[SQL ExecTx] [%s]\t Begin Tx Error:[%v]", time.Since(start), err)
		return err
	}
	txWrap := _TxWrap{
		start:           start,
		tx:              tx,
		execSQLSequence: 1,
		db:              c,
	}
	defer func() {
		if p := recover(); p != nil {
			_ = txWrap.rollback(fmt.Errorf("panic:%v", p))
			panic(p)
		}
	}()
	if err := bizFunc(&txWrap); err != nil {
		_ = txWrap.rollback(err)
		return err
	}
	return txWrap.commit()
}

// Get 缺省DB查询
func Get(dbModel interface{}, strSQL string, args ...interface{}) (int, error) {
	if defaultDatabase == nil {
		return 0, fmt.Errorf("no default db,please use NewDefaultDatabase method to set")
	}
	return defaultDatabase.query(dbModel, strSQL, true, nil, nil, args...)
}

func Get2(dbModel interface{}, strSQL string, logArgs func(args ...interface{}) string, logResult func(r interface{}) string, args ...interface{}) (int, error) {
	if defaultDatabase == nil {
		return 0, fmt.Errorf("no default db,please use NewDefaultDatabase method to set")
	}
	return defaultDatabase.query(dbModel, strSQL, true, logArgs, logResult, args...)
}

// Gets 缺省DB查询
func Gets(dbModel interface{}, strSQL string, args ...interface{}) (int, error) {
	if defaultDatabase == nil {
		return 0, fmt.Errorf("no default db,please use NewDefaultDatabase method to set")
	}
	return defaultDatabase.query(dbModel, strSQL, false, nil, nil, args...)
}
func Gets2(dbModel interface{}, strSQL string, logArgs func(args ...interface{}) string, args ...interface{}) (int, error) {
	if defaultDatabase == nil {
		return 0, fmt.Errorf("no default db,please use NewDefaultDatabase method to set")
	}
	return defaultDatabase.query(dbModel, strSQL, false, logArgs, nil, args...)
}

// Exec 缺省DB执行SQL
func Exec(strSQL string, args ...interface{}) (int, error) {
	if defaultDatabase == nil {
		return 0, fmt.Errorf("no default db,please use NewDefaultDatabase method to set")
	}
	return defaultDatabase.Exec(strSQL, args...)
}

func Exec2(strSQL string, logArgs func(args ...interface{}) string, args ...interface{}) (int, error) {
	if defaultDatabase == nil {
		return 0, fmt.Errorf("no default db,please use NewDefaultDatabase method to set")
	}
	return defaultDatabase.Exec2(strSQL, logArgs, args...)
}

// Insert 缺省DB执行Insert SQL
func Insert(strSQL string, args ...interface{}) (int, error) {
	if defaultDatabase == nil {
		return 0, fmt.Errorf("no default db,please use NewDefaultDatabase method to set")
	}
	return defaultDatabase.Insert(strSQL, args...)
}

func Insert2(strSQL string, logArgs func(args ...interface{}) string, args ...interface{}) (int, error) {
	if defaultDatabase == nil {
		return 0, fmt.Errorf("no default db,please use NewDefaultDatabase method to set")
	}
	return defaultDatabase.Insert2(strSQL, logArgs, args...)
}

// GetWhereClause 构建SQL条件从句,不含where关键字，只支持string,int及其数组
func GetWhereClause(searchColumns map[string]interface{}) string {
	strWhereValue := ""
	for searchColumnName, v := range searchColumns {
		var strInValues []string
		switch searchColumnValues := v.(type) {
		case []string:
			for _, s := range searchColumnValues {
				v := strings.Replace(s, mysqlEscapeBackslash, mysqlEscapeBackslashReplaced, -1)
				v = strings.Replace(v, mysqlEscapeSingleQuote, mysqlEscapeSingleQuoteReplaced, -1)
				strInValues = append(strInValues, fmt.Sprintf("'%s'", v))
			}
		case []int:
			for _, s := range searchColumnValues {
				strInValues = append(strInValues, fmt.Sprintf("%d", s))
			}
		case string:
			v := strings.Replace(searchColumnValues, mysqlEscapeBackslash, mysqlEscapeBackslashReplaced, -1)
			v = strings.Replace(v, mysqlEscapeSingleQuote, mysqlEscapeSingleQuoteReplaced, -1)
			strInValues = append(strInValues, fmt.Sprintf("'%s'", v))
		case int:
			strInValues = append(strInValues, fmt.Sprintf("%d", searchColumnValues))
		}
		if countValues := len(strInValues); countValues > 0 {
			c := ""
			if countValues == 1 {
				c = fmt.Sprintf("%s=%s", searchColumnName, strInValues[0])
			} else {
				c = fmt.Sprintf("%s IN (%s) ", searchColumnName, strings.Join(strInValues, ","))
			}
			if strWhereValue == "" {
				strWhereValue = c
			} else {
				strWhereValue = fmt.Sprintf("%s AND %s", strWhereValue, c)
			}
		}
	}
	return strWhereValue
}

// Exec 执行SQL，返回影响行数，最后InsertID
func (c *_TxWrap) Exec(strSQL string, args ...interface{}) (int, int, error) {
	start := time.Now()
	defer func() {
		c.execSQLSequence++
	}()
	result, err := c.tx.Exec(strSQL, args...)
	if err != nil {
		log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s] [%s] Args [%v] Error:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, args, err)
		return 0, 0, err
	}
	rowsCount, affectedError := result.RowsAffected()
	if affectedError != nil {
		rowsCount = -1
	}
	insertID, insertError := result.LastInsertId()
	if insertError != nil {
		insertID = -1
	}
	if c.db.writeLog && log.IsEnableCategoryInfoLog("SQL") {
		log.Info2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s] [%s] Args [%v] [Affected Row Counts=%d]", c, c.execSQLSequence, time.Since(start), strSQL, args, rowsCount)
	}
	return int(rowsCount), int(insertID), nil
}

func (c *_TxWrap) ScanGet(strSQL string, dest ...interface{}) (int, error) {
	start := time.Now()
	defer func() {
		c.execSQLSequence++
	}()
	result := c.tx.QueryRow(strSQL)
	err := result.Err()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\t[row count=0]", c, c.execSQLSequence, time.Since(start), strSQL, "")
			return 0, nil
		}
		log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tError:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, err)
		return 0, err
	}
	if err := result.Scan(dest...); err != nil {
		log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tError:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, err)
		return 0, err
	}
	if c.db.writeLog && log.IsEnableCategoryInfoLog("SQL") {
		log.Info2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\t[%v]", c, c.execSQLSequence, time.Since(start), strSQL, dest)
	}
	return 1, nil
}

func (c *_TxWrap) Get(dbModel interface{}, strSQL string, args ...interface{}) (int, error) {
	start := time.Now()
	value := reflect.ValueOf(dbModel)
	if value.Kind() != reflect.Ptr {
		return 0, errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if value.IsNil() {
		return 0, errors.New("nil pointer passed to StructScan destination")
	}
	defer func() {
		c.execSQLSequence++
	}()
	result, err := c.tx.Query(strSQL, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\t[row count=0]", c, c.execSQLSequence, time.Since(start), strSQL, args)
			return 0, nil
		}
		log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\tError:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, args, err)
		return 0, err
	}
	if result.Next() {
		columns, err := result.ColumnTypes()
		if err != nil {
			log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\tError:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, args, err)
			_ = result.Close()
			return 0, err
		}
		var columnValue []interface{}
		columnIndex := make(map[string]int)
		for i, d := range columns {
			columnIndex[d.Name()] = i
			columnValue = append(columnValue, new(string))
		}
		if err := result.Scan(columnValue...); err != nil {
			log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\tError:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, args, err)
			_ = result.Close()
			return 0, err
		}
		v := value.Elem()
		t := v.Type()
		if t.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				fv := v.Field(i)
				if dbName, ok := t.Field(i).Tag.Lookup("db"); ok && dbName != "" && fv.CanSet() {
					if index, ok := columnIndex[dbName]; ok {
						if val := columnValue[index].(*string); *val != "" {
							switch fv.Kind() {
							case reflect.String:
								fv.SetString(*val)
							case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
								if iVal, err := strconv.Atoi(*val); err == nil {
									fv.SetInt(int64(iVal))
								}
							case reflect.Float32, reflect.Float64:
								if fVal, err := strconv.ParseFloat(*val, 64); err == nil {
									fv.SetFloat(fVal)
								}
							case reflect.Uint:
								if iVal, err := strconv.Atoi(*val); err == nil {
									fv.SetUint(uint64(iVal))
								}
							case reflect.Bool:
								bv := strings.ToLower(*val)
								switch bv {
								case "1", "true":
									fv.SetBool(true)
								case "0", "false":
									fv.SetBool(false)
								default:
									if bv != "" {
										fv.SetBool(true)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	if c.db.writeLog && log.IsEnableCategoryInfoLog("SQL") {
		log.Info2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\t[%v]", c, c.execSQLSequence, time.Since(start), strSQL, args, dbModel)
	}
	_ = result.Close()
	return 1, nil
}

func (c *_TxWrap) Gets(dbModel interface{}, strSQL string, args ...interface{}) (int, error) {
	start := time.Now()
	value := reflect.ValueOf(dbModel)
	if value.Kind() != reflect.Ptr {
		return 0, errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if value.IsNil() {
		return 0, errors.New("nil pointer passed to StructScan destination")
	}
	direct := reflect.Indirect(value)
	slice, err := _baseType(value.Type(), reflect.Slice)
	if err != nil {
		return 0, err
	}
	isPtr := slice.Elem().Kind() == reflect.Ptr
	base := reflectx.Deref(slice.Elem())
	defer func() {
		c.execSQLSequence++
	}()
	result, err := c.tx.Query(strSQL, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\t[row count=0]", c, c.execSQLSequence, time.Since(start), strSQL, args)
			return 0, nil
		}
		log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\tError:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, args, err)
		return 0, err
	}
	columns, err := result.ColumnTypes()
	if err != nil {
		log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\tError:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, args, err)
		_ = result.Close()
		return 0, err
	}
	columnIndex := make(map[string]int)
	for i, d := range columns {
		columnIndex[d.Name()] = i
	}
	var vp reflect.Value
	rowCounts := 0
	for result.Next() {
		vp = reflect.New(base)
		var columnValue []interface{}
		for i, j := 0, len(columns); i < j; i++ {
			columnValue = append(columnValue, new(string))
		}
		if err := result.Scan(columnValue...); err != nil {
			log.Error2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\tError:[%v]", c, c.execSQLSequence, time.Since(start), strSQL, args, err)
			_ = result.Close()
			return 0, err
		}
		v := vp.Elem()
		t := v.Type()
		if t.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				fv := v.Field(i)
				if dbName, ok := t.Field(i).Tag.Lookup("db"); ok && dbName != "" && fv.CanSet() {
					if index, ok := columnIndex[dbName]; ok {
						if val := columnValue[index].(*string); *val != "" {
							switch fv.Kind() {
							case reflect.String:
								fv.SetString(*val)
							case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
								if iVal, err := strconv.Atoi(*val); err == nil {
									fv.SetInt(int64(iVal))
								}
							case reflect.Float32, reflect.Float64:
								if fVal, err := strconv.ParseFloat(*val, 64); err == nil {
									fv.SetFloat(fVal)
								}
							case reflect.Uint:
								if iVal, err := strconv.Atoi(*val); err == nil {
									fv.SetUint(uint64(iVal))
								}
							case reflect.Bool:
								bv := strings.ToLower(*val)
								switch bv {
								case "1", "true":
									fv.SetBool(true)
								case "0", "false":
									fv.SetBool(false)
								default:
									if bv != "" {
										fv.SetBool(true)
									}
								}
							}
						}
					}
				}
			}
		}
		if isPtr {
			direct.Set(reflect.Append(direct, vp))
		} else {
			direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
		}
		rowCounts++
	}
	if c.db.writeLog && log.IsEnableCategoryInfoLog("SQL") {
		log.Info2(c.db.logger, "[SQL ExecTx] [%p] [Execute SQL Sequence:%d] [%s]\t[%s]\tArgs [%v]\t[row count=%d]", c, c.execSQLSequence, time.Since(start), strSQL, args, rowCounts)
	}
	return rowCounts, nil
}

func (c *_TxWrap) rollback(logError error) error {
	err := c.tx.Rollback()
	if err != nil {
		log.Error2(c.db.logger, "[SQL ExecTx] [%p] Rollback error:[%v] ", c, err)
	}
	log.Error2(c.db.logger, "[SQL ExecTx] [%p] [%s]\tBizFunc error:[%v] rollback", c, time.Since(c.start), logError)
	return err
}

func (c *_TxWrap) commit() error {
	err := c.tx.Commit()
	if err != nil {
		log.Error2(c.db.logger, "[SQL ExecTx] [%p] [%s]\tCommit error:[%v] ", c, time.Since(c.start), err)
	} else {
		if c.db.writeLog && log.IsEnableCategoryInfoLog("SQL") {
			log.Info2(c.db.logger, "[SQL ExecTx] [%p] [%s]\tCommit success", c, time.Since(c.start))
		}
	}
	return err
}

func _baseType(t reflect.Type, expected reflect.Kind) (reflect.Type, error) {
	t = reflectx.Deref(t)
	if t.Kind() != expected {
		return nil, fmt.Errorf("expected %s but got %s", expected, t.Kind())
	}
	return t, nil
}
