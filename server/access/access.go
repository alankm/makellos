package access

import (
	"database/sql"
	"errors"

	"gopkg.in/inconshreveable/log15.v2"
)

func New(configs map[string]string, log log15.Logger, db *sql.DB) (Access, error) {
	if configs == nil {
		return nil, errors.New("nil configs")
	}
	var val string
	var ok bool
	if val, ok = configs["type"]; !ok {
		return nil, errors.New("configs don't specify type")
	}
	switch val {
	case "local":
		local := new(Local)
		err := local.setup(log, db)
		if err != nil {
			return nil, err
		}
		return local, nil
	default:
		return nil, errors.New("type not supported")
	}
}
