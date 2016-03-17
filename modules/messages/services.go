package messages

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/alankm/makellos/core/shared"
	"github.com/gorilla/websocket"
)

func (m *Messages) httpDebug(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, shared.Debug)
}

func (m *Messages) httpInfo(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, shared.Info)
}

func (m *Messages) httpWarn(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, shared.Warn)
}

func (m *Messages) httpError(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, shared.Error)
}

func (m *Messages) httpCrit(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, shared.Crit)
}

func (m *Messages) httpAlert(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, shared.Alert)
}

func (m *Messages) httpAll(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, shared.All)
}

func (m *Messages) wsDebug(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, shared.Debug)
}

func (m *Messages) wsInfo(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, shared.Info)
}

func (m *Messages) wsWarn(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, shared.Warn)
}

func (m *Messages) wsError(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, shared.Error)
}

func (m *Messages) wsCrit(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, shared.Crit)
}

func (m *Messages) wsAlert(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, shared.Alert)
}

func (m *Messages) wsAll(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, shared.All)
}

func (m *Messages) post(s shared.Session, w http.ResponseWriter, r *http.Request) {
	var sev, code, message string
	var sevno shared.Severity
	var ok bool
	// Severity
	if val, ok := r.Header["Severity"]; ok && len(val) == 1 {
		sev = val[0]
	}
	if sevno, ok = shared.SeverityCodes[sev]; !ok {
		w.Write(shared.NewFailResponse(0, "missing or bad severity header").JSON())
		return
	}
	// Code
	if val, ok := r.Header["Code"]; ok && len(val) == 1 && val[0] != "" {
		code = val[0]
	} else {
		w.Write(shared.NewFailResponse(0, "missing or bad code header").JSON())
		return
	}
	// Message
	if val, ok := r.Header["Message"]; ok && len(val) == 1 && val[0] != "" {
		message = val[0]
	} else {
		w.Write(shared.NewFailResponse(0, "missing or bad message header").JSON())
		return
	}
	// Args
	var args = make(map[string]string)
	var argsArray = r.Header["Args"]
	if len(args)%2 == 1 {
		w.Write(shared.NewFailResponse(0, "bad args header").JSON())
		return
	}
	for i := 0; i < len(argsArray); i = i + 2 {
		args[argsArray[i]] = argsArray[i+1]
	}

	log := &shared.Log{
		Severity: sevno,
		Time:     time.Now().Unix(),
		Rules: shared.Rules{
			Owner: s.Username(),
			Group: s.GID(),
			Mode:  s.Mode(),
		},
		Code:    code,
		Message: message,
		Args:    args,
	}
	m.postLog(log)
	w.Write(shared.Success.JSON())
}

func (m *Messages) ws(s shared.Session, w http.ResponseWriter, r *http.Request, severity shared.Severity) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.Write(shared.ResponseVorteilInternal.JSON())
		return
	}
	// goroutine to health-check websocket
	connMonitor := make(chan bool)
	go func(c *websocket.Conn, monitor chan bool) {
		for {
			if _, _, err := c.NextReader(); err != nil {
				monitor <- true
				c.Close()
				break
			}
		}
	}(conn, connMonitor)
	// Vorteil leader status health-check
	leaderMonitor := make(chan bool)
	m.core.NotifyLeaderChange(&leaderMonitor)
	defer m.core.UnsubscribeLeaderChange(&leaderMonitor)
	// listen for more messages
	monitor := make(chan *shared.Log)
	m.clients[severity][monitor] = true
	defer delete(m.clients[severity], monitor)
	for {
		select {
		case <-leaderMonitor:
			return
		case x := <-monitor:
			if s.CanRead(&x.Rules) {
				ao := message{
					Time:     x.Time,
					Message:  x.Message,
					Code:     x.Code,
					Args:     x.Args,
					Severity: shared.SeverityStrings[x.Severity],
				}
				err := conn.WriteJSON(ao)
				if err != nil {
					return
				}
			}
		case <-connMonitor:
			return
		}
	}
}

func sqlSevString(severity shared.Severity) string {
	sevString := "("
	switch severity {
	case shared.Alert:
		sevString += strconv.Itoa(shared.Warn)
		sevString += ", "
		sevString += strconv.Itoa(shared.Error)
		sevString += ", "
		sevString += strconv.Itoa(shared.Crit)
	case shared.All:
		sevString += strconv.Itoa(shared.Debug)
		sevString += ", "
		sevString += strconv.Itoa(shared.Info)
		sevString += ", "
		sevString += strconv.Itoa(shared.Warn)
		sevString += ", "
		sevString += strconv.Itoa(shared.Error)
		sevString += ", "
		sevString += strconv.Itoa(shared.Crit)
	default:
		sevString += strconv.Itoa(int(severity))
	}
	sevString += ")"
	return sevString
}

func sqlGroupsString(groups []string) string {
	groupsString := "("
	for i, group := range groups {
		groupsString += "'" + group + "'"
		if i < len(groups)-1 {
			groupsString += ", "
		}
	}
	groupsString += ")"
	return groupsString
}

func (m *Messages) http(s shared.Session, w http.ResponseWriter, r *http.Request, severity shared.Severity) {
	var offset, length int
	var start, end int64
	var sort string
	// offset
	vals := r.URL.Query()
	val, ok := vals["offset"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 32)
		offset = int(x)
	}
	// length
	val, ok = vals["length"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 32)
		length = int(x)
	} else {
		length = -1
	}
	// start
	val, ok = vals["start"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 64)
		start = int64(x)
	}
	// end
	val, ok = vals["end"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 64)
		end = int64(x)
	} else {
		end = time.Now().Unix()
	}
	// sort
	val, ok = vals["sort"]
	if ok && len(val) > 0 && val[0] == "true" {
		sort = "severity DESC, id DESC"
	} else {
		sort = "id DESC"
	}

	// build strings for SQL request
	sevString := sqlSevString(severity)
	groupsString := sqlGroupsString(s.Groups())

	// Query
	var rows *sql.Rows
	var err error
	var count int
	if s.SU() {
		rows, err = m.db.Query("SELECT * FROM messages WHERE (time BETWEEN ? AND ?) AND (sev IN "+sevString+") ORDER BY "+sort+" LIMIT ?,?", start, end, offset, length)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		row := m.db.QueryRow("SELECT COUNT(*) FROM messages WHERE (time BETWEEN ? AND ?) AND (sev IN "+sevString+")", start, end)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		err = row.Scan(&count)
		if err != nil {
			panic(err)
		}
	} else {
		rows, err = m.db.Query("SELECT * FROM messages WHERE (time BETWEEN ? AND ?) AND (sev IN "+sevString+") AND ((own = ? AND (mode & 0x100) = 0x100) OR (own != ? AND grp IN "+groupsString+" AND (mode & 0x020) = 0x020) OR (own != ? AND grp NOT IN "+groupsString+" AND (mode & 0x004) = 0x004)) ORDER BY "+sort+" LIMIT ?,?", start, end, s.Username(), s.Username(), s.Username(), offset, length)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		row := m.db.QueryRow("SELECT COUNT(*) FROM messages WHERE (time BETWEEN ? AND ?) AND (sev IN "+sevString+") AND ((own = ? AND (mode & 0x100) = 0x100) OR (own != ? AND grp IN "+groupsString+" AND (mode & 0x020) = 0x020) OR (own != ? AND grp NOT IN "+groupsString+" AND (mode & 0x004) = 0x004))", start, end, s.Username(), s.Username(), s.Username())
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		err = row.Scan(&count)
		if err != nil {
			panic(err)
		}
	}

	var out []message
	for rows.Next() {
		var id, time, severity, mode int
		var msg, code string
		var owner, group string
		err = rows.Scan(&id, &time, &severity, &msg, &code, &owner, &group, &mode)
		if err != nil {
			panic(err)
		}

		ao := message{
			Time:     int64(time),
			Message:  msg,
			Code:     code,
			Args:     make(map[string]string),
			Severity: shared.SeverityStrings[shared.Severity(severity)],
		}

		argrows, err := m.db.Query("SELECT * FROM args WHERE id=?", id)
		if err != nil {
			panic(err)
		}
		defer argrows.Close()

		for argrows.Next() {
			var rid int
			var key, value string
			err = argrows.Scan(&rid, &key, &value)
			if err != nil {
				panic(err)
			}
			ao.Args[key] = value
		}

		out = append(out, ao)
	}

	response := shared.NewSuccessResponse(&listPayload{count, out})
	w.Write(response.JSON())
}
