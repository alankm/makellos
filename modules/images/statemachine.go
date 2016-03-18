package images

import (
	"net/url"
	"strconv"
	"time"

	"github.com/alankm/makellos/core/shared"
)

var emptyPayloadChecksum = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

type smResponse struct {
	Success  bool
	Response []byte
	Delete   []string
	Logs     []shared.Log
}

func (i *Images) apply(args []byte) []byte {
	var err error
	resp := new(smResponse)
	req := new(Request)
	i.core.Decode(args, req)
	req.s, err = i.core.HashedLogin(req.Username, req.Hashword)
	if err != nil {
		return i.core.Encode(resp)
	}

	switch req.Method {
	case "POST":
		return i.doPost(req, resp)
	case "PUT":
		return i.doPut(req, resp)
	case "DELETE":
		return i.doDelete(req, resp)
	default:
		return i.core.Encode(resp)
	}

}

func (i *Images) doPost(r *Request, resp *smResponse) []byte {
	var err error
	row := i.db.QueryRow("SELECT COUNT(*) FROM images WHERE checksum=?", r.Checksum)
	var x int
	err = row.Scan(&x)
	if err != nil {
		panic(err)
	}
	if x == 0 {
		resp.Delete = append(resp.Delete, r.Checksum)
	}

	rec := i.lookup(r.Path)
	parent, name, _ := splitPath(r.Path)
	auth := ""
	date := int64(0)
	desc := ""
	if val, ok := r.Header["Author"]; ok && len(val) == 1 {
		auth = val[0]
	}
	if val, ok := r.Header["Description"]; ok && len(val) == 1 {
		desc = val[0]
	}
	if val, ok := r.Header["Time"]; ok && len(val) == 1 {
		date, _ = strconv.ParseInt(val[0], 10, 64)
	}
	if rec != nil {
		u, err := url.Parse(r.URL)
		if err != nil {
			panic(err)
		}
		if val, ok := u.Query()["overwrite"]; !ok || len(val) != 1 || val[0] != "true" {
			// no overwrite parameter supplied
			return i.core.Encode(resp)
		}
		// overwrite logic here
		rows, count := i.lookupChildren(r.Path, -1, 0)
		rows.Close()
		if count > 0 {
			// recursive overwrites are not supported.
			return i.core.Encode(resp)
		}
		_, err = i.db.Exec("UPDATE images SET checksum=?, date=?, author=?, description=? WHERE parent=? AND name=?", r.Checksum, rec.date, rec.auth, rec.desc, parent, name)
		if err != nil {
			panic(err)
		}
		resp.Success = true
		resp.Delete = make([]string, 0)
		row := i.db.QueryRow("SELECT COUNT(*) FROM images WHERE checksum=?", rec.chk)
		var x int
		err = row.Scan(&x)
		if err != nil {
			panic(err)
		}
		if x == 0 {
			resp.Delete = append(resp.Delete, rec.chk)
		}
		resp.Logs = append(resp.Logs, shared.Log{
			Severity: shared.Info,
			Time:     time.Now().Unix(),
			Rules: shared.Rules{
				Owner: r.s.Username(),
				Group: r.s.GID(),
				Mode:  r.s.Mode(),
			},
			Code:    "images.NewFolder",
			Message: "a folder was added to images",
			Args: map[string]string{
				"name":     name,
				"location": parent,
			},
		})
		return i.core.Encode(resp)
	}

	// non-overwrite logic here
	folder := r.Checksum == emptyPayloadChecksum
	if folder {
		// new folder
		_, err = i.db.Exec("INSERT INTO images (name, parent, directory, checksum, date, author, description) VALUES (?,?,?,?,?,?,?)", name, parent, true, r.Checksum, date, auth, desc)
		if err != nil {
			return i.core.Encode(resp)
		}
		resp.Success = true
		resp.Logs = append(resp.Logs, shared.Log{
			Severity: shared.Info,
			Time:     time.Now().Unix(),
			Rules: shared.Rules{
				Owner: r.s.Username(),
				Group: r.s.GID(),
				Mode:  r.s.Mode(),
			},
			Code:    "images.NewFolder",
			Message: "a folder was added to images",
			Args: map[string]string{
				"name":     name,
				"location": parent,
			},
		})
		return i.core.Encode(resp)
	}

	// new image
	_, err = i.db.Exec("INSERT INTO images (name, parent, directory, checksum, date, author, description) VALUES (?,?,?,?,?,?,?)", name, parent, false, r.Checksum, date, auth, desc)
	if err != nil {
		return i.core.Encode(resp)
	}
	resp.Success = true
	resp.Delete = make([]string, 0)
	resp.Logs = append(resp.Logs, shared.Log{
		Severity: shared.Info,
		Time:     time.Now().Unix(),
		Rules: shared.Rules{
			Owner: r.s.Username(),
			Group: r.s.GID(),
			Mode:  r.s.Mode(),
		},
		Code:    "images.NewImage",
		Message: "an image was added to images",
		Args: map[string]string{
			"name":     name,
			"location": parent,
			"checksum": r.Checksum,
		},
	})
	return i.core.Encode(resp)
}

func (i *Images) doPut(r *Request, resp *smResponse) []byte {
	rec := i.lookup(r.Path)
	if rec == nil {
		// no such record to update
		return i.core.Encode(resp)
	}

	if val, ok := r.Header["Author"]; ok && len(val) == 1 {
		rec.auth = val[0]
	}
	if val, ok := r.Header["Description"]; ok && len(val) == 1 {
		rec.desc = val[0]
	}
	if val, ok := r.Header["Time"]; ok && len(val) == 1 {
		date, err := strconv.ParseInt(val[0], 10, 64)
		if err == nil {
			rec.date = date
		}
	}

	parent, name, _ := splitPath(r.Path)
	_, err := i.db.Exec("UPDATE images SET date=?, author=?, description=? WHERE parent=? AND name=?", rec.date, rec.auth, rec.desc, parent, name)
	if err != nil {
		panic(err)
	}
	resp.Success = true
	resp.Logs = append(resp.Logs, shared.Log{
		Severity: shared.Info,
		Time:     time.Now().Unix(),
		Rules: shared.Rules{
			Owner: r.s.Username(),
			Group: r.s.GID(),
			Mode:  r.s.Mode(),
		},
		Code:    "images.Attributes",
		Message: "an object's attributes were updated",
		Args: map[string]string{
			"name":     name,
			"location": parent,
		},
	})
	return i.core.Encode(resp)
}

func (i *Images) doDelete(r *Request, resp *smResponse) []byte {
	rec := i.lookup(r.Path)
	if rec == nil {
		// no such record to delete
		return i.core.Encode(resp)
	}
	typeStr := "file"
	if rec.dir {
		typeStr = "folder"
	}
	rows, count := i.lookupChildren(r.Path, -1, 0)
	rows.Close()
	if count != 0 {
		// not supporting recursive changes
		return i.core.Encode(resp)
	}
	parent, name, _ := splitPath(r.Path)
	_, err := i.db.Exec("DELETE FROM images WHERE parent=? AND name=?", parent, name)
	if err != nil {
		panic(err)
	}
	resp.Success = true
	resp.Logs = append(resp.Logs, shared.Log{
		Severity: shared.Info,
		Time:     time.Now().Unix(),
		Rules: shared.Rules{
			Owner: r.s.Username(),
			Group: r.s.GID(),
			Mode:  r.s.Mode(),
		},
		Code:    "images.Delete",
		Message: "an object was deleted from images",
		Args: map[string]string{
			"name":     name,
			"location": parent,
			"type":     typeStr,
		},
	})
	if !rec.dir {
		row := i.db.QueryRow("SELECT COUNT(*) FROM images WHERE checksum=?", rec.chk)
		var x int
		err = row.Scan(&x)
		if err != nil {
			panic(err)
		}
		if x == 0 {
			resp.Delete = append(resp.Delete, rec.chk)
		}
	}
	return i.core.Encode(resp)
}
