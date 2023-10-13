package data

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	digits01                  = "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789"
	digits10                  = "0000000000111111111122222222223333333333444444444455555555556666666666777777777788888888889999999999"
	DefaultMySQLMaxPacketSize = 1<<24 - 1
)

type (
	// MySQLInterpolate sql语句参数内插器
	MySQLInterpolate struct {
		maxAllowedPacket  int
		escapeUsingQuotes bool
	}
)

var (
	defaultMySQLInterpolate *MySQLInterpolate
)

// NewMySQLInterpolate MySQLInterpolate
func NewMySQLInterpolate(escapeUsingQuotes bool, maxAllowedPacket int) *MySQLInterpolate {
	mi := &MySQLInterpolate{
		maxAllowedPacket:  maxAllowedPacket,
		escapeUsingQuotes: escapeUsingQuotes,
	}
	return mi
}

func init() {
	defaultMySQLInterpolate = NewMySQLInterpolate(false, DefaultMySQLMaxPacketSize)
}

// GetBuiltSQL 处理SQL,得到绑定后可以执行的SQL
func GetBuiltSQL(strSQL string, args ...interface{}) (string, error) {
	return defaultMySQLInterpolate.GetBuiltSQL(strSQL, args...)
}

func (c *MySQLInterpolate) reserveBuffer(buf []byte, appendSize int) []byte {
	newSize := len(buf) + appendSize
	if cap(buf) < newSize {
		// Grow buffer exponentially
		newBuf := make([]byte, len(buf)*2+appendSize)
		copy(newBuf, buf)
		buf = newBuf
	}
	return buf[:newSize]
}

func (c *MySQLInterpolate) escapeBytesQuotes(buf, v []byte) []byte {
	pos := len(buf)
	buf = c.reserveBuffer(buf, len(v)*2)
	for _, c := range v {
		if c == '\'' {
			buf[pos] = '\''
			buf[pos+1] = '\''
			pos += 2
		} else {
			buf[pos] = c
			pos++
		}
	}
	return buf[:pos]
}

func (c *MySQLInterpolate) escapeBytesBackslash(buf, v []byte) []byte {
	pos := len(buf)
	buf = c.reserveBuffer(buf, len(v)*2)
	for _, c := range v {
		switch c {
		case '\x00':
			buf[pos] = '\\'
			buf[pos+1] = '0'
			pos += 2
		case '\n':
			buf[pos] = '\\'
			buf[pos+1] = 'n'
			pos += 2
		case '\r':
			buf[pos] = '\\'
			buf[pos+1] = 'r'
			pos += 2
		case '\x1a':
			buf[pos] = '\\'
			buf[pos+1] = 'Z'
			pos += 2
		case '\'':
			buf[pos] = '\\'
			buf[pos+1] = '\''
			pos += 2
		case '"':
			buf[pos] = '\\'
			buf[pos+1] = '"'
			pos += 2
		case '\\':
			buf[pos] = '\\'
			buf[pos+1] = '\\'
			pos += 2
		default:
			buf[pos] = c
			pos++
		}
	}
	return buf[:pos]
}

func (c *MySQLInterpolate) escapeStringBackslash(buf []byte, v string) []byte {
	pos := len(buf)
	buf = c.reserveBuffer(buf, len(v)*2)
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch c {
		case '\x00':
			buf[pos] = '\\'
			buf[pos+1] = '0'
			pos += 2
		case '\n':
			buf[pos] = '\\'
			buf[pos+1] = 'n'
			pos += 2
		case '\r':
			buf[pos] = '\\'
			buf[pos+1] = 'r'
			pos += 2
		case '\x1a':
			buf[pos] = '\\'
			buf[pos+1] = 'Z'
			pos += 2
		case '\'':
			buf[pos] = '\\'
			buf[pos+1] = '\''
			pos += 2
		case '"':
			buf[pos] = '\\'
			buf[pos+1] = '"'
			pos += 2
		case '\\':
			buf[pos] = '\\'
			buf[pos+1] = '\\'
			pos += 2
		default:
			buf[pos] = c
			pos++
		}
	}
	return buf[:pos]
}

func (c *MySQLInterpolate) escapeStringQuotes(buf []byte, v string) []byte {
	pos := len(buf)
	buf = c.reserveBuffer(buf, len(v)*2)
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c == '\'' {
			buf[pos] = '\''
			buf[pos+1] = '\''
			pos += 2
		} else {
			buf[pos] = c
			pos++
		}
	}
	return buf[:pos]
}

// GetBuiltSQL 得到绑定后的SQL语句
func (c *MySQLInterpolate) GetBuiltSQL(query string, args ...interface{}) (string, error) {
	if strings.Count(query, "?") != len(args) {
		return "", fmt.Errorf("placehoder and args unmatched")
	}
	buf := make([]byte, 4096)
	if buf == nil {
		return "", fmt.Errorf("failure to allocate buffer")
	}
	buf = buf[:0]
	argPos := 0

	for i := 0; i < len(query); i++ {
		q := strings.IndexByte(query[i:], '?')
		if q == -1 {
			buf = append(buf, query[i:]...)
			break
		}
		buf = append(buf, query[i:i+q]...)
		i += q

		arg := args[argPos]
		argPos++

		if arg == nil {
			buf = append(buf, "NULL"...)
			continue
		}

		switch v := arg.(type) {
		case int8:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int16:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int32:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int64:
			buf = strconv.AppendInt(buf, v, 10)
		case float32:
			buf = strconv.AppendFloat(buf, float64(v), 'g', -1, 64)
		case float64:
			buf = strconv.AppendFloat(buf, v, 'g', -1, 64)
		case bool:
			if v {
				buf = append(buf, '1')
			} else {
				buf = append(buf, '0')
			}
		case time.Time:
			if v.IsZero() {
				buf = append(buf, "'0000-00-00'"...)
			} else {
				v := v.In(time.Local)
				v = v.Add(time.Nanosecond * 500) // To round under microsecond
				year := v.Year()
				year100 := year / 100
				year1 := year % 100
				month := v.Month()
				day := v.Day()
				hour := v.Hour()
				minute := v.Minute()
				second := v.Second()
				micro := v.Nanosecond() / 1000

				buf = append(buf, []byte{
					'\'',
					digits10[year100], digits01[year100],
					digits10[year1], digits01[year1],
					'-',
					digits10[month], digits01[month],
					'-',
					digits10[day], digits01[day],
					' ',
					digits10[hour], digits01[hour],
					':',
					digits10[minute], digits01[minute],
					':',
					digits10[second], digits01[second],
				}...)

				if micro != 0 {
					micro10000 := micro / 10000
					micro100 := micro / 100 % 100
					micro1 := micro % 100
					buf = append(buf, []byte{
						'.',
						digits10[micro10000], digits01[micro10000],
						digits10[micro100], digits01[micro100],
						digits10[micro1], digits01[micro1],
					}...)
				}
				buf = append(buf, '\'')
			}
		case []byte:
			if v == nil {
				buf = append(buf, "NULL"...)
			} else {
				buf = append(buf, "_binary'"...)
				if c.escapeUsingQuotes {
					buf = c.escapeBytesQuotes(buf, v)
				} else {
					buf = c.escapeBytesBackslash(buf, v)
				}
				buf = append(buf, '\'')
			}
		case string:
			buf = append(buf, '\'')
			if c.escapeUsingQuotes {
				buf = c.escapeStringQuotes(buf, v)
			} else {
				buf = c.escapeStringBackslash(buf, v)
			}
			buf = append(buf, '\'')
		default:
			return "", fmt.Errorf("unsupport field type [%T]", v)
		}
		if len(buf)+4 > c.maxAllowedPacket {
			return "", fmt.Errorf("sql length too large")
		}
	}
	if argPos != len(args) {
		return "", fmt.Errorf("placehoder and args unmatched")
	}
	return string(buf), nil
}
