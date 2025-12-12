package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

func GetProjectIDByName(name string) (int, error) {
	d, err := GetDB()
	if err != nil {
		return 0, err
	}
	defer d.Close()

	var id int
	err = d.QueryRow("SELECT id FROM projects WHERE name = ? LIMIT 1", name).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no project named %q", name)
	}
	return id, err
}

func ResolveProjectIdentifier(ident string) (int, error) {
	if ident == "" {
		return 0, errors.New("empty project identifier")
	}
	if id, err := strconv.Atoi(ident); err == nil {
		return id, nil
	}
	return GetProjectIDByName(ident)
}

func GetSiteIDByName(name string) (int, error) {
	d, err := GetDB()
	if err != nil {
		return 0, err
	}
	defer d.Close()

	var id int
	err = d.QueryRow("SELECT id FROM sites WHERE name = ? LIMIT 1", name).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no site named %q", name)
	}
	return id, err
}

func ResolveSiteIdentifier(ident string) (int, error) {
	if ident == "" {
		return 0, errors.New("empty site identifier")
	}
	if id, err := strconv.Atoi(ident); err == nil {
		return id, nil
	}
	return GetSiteIDByName(ident)
}

func GetLoggerIDByName(name string) (int, error) {
	d, err := GetDB()
	if err != nil {
		return 0, err
	}
	defer d.Close()

	var id int
	err = d.QueryRow("SELECT id FROM loggers WHERE name = ? LIMIT 1", name).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no logger named %q", name)
	}
	return id, err
}

func ResolveLoggerIdentifier(ident string) (int, error) {
	if ident == "" {
		return 0, errors.New("empty logger identifier")
	}
	if id, err := strconv.Atoi(ident); err == nil {
		return id, nil
	}
	return GetLoggerIDByName(ident)
}
