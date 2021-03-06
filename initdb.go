package main

import (
	"github.com/semog/go-sqldb"
)

func (st *sqlStore) Init(databaseFile string) error {
	var err error
	st.db, err = sqldb.OpenAndPatchDb(databaseFile, dbPatchFuncs)
	return err
}

// The array of patch functions that will automatically upgrade the database.
var dbPatchFuncs = []sqldb.PatchFuncType{
	// Add new patch functions to this array to automatically upgrade the database.
	{PatchID: 1, PatchFunc: func(sdb *sqldb.SQLDb) error {
		if err := sdb.CreateTable(`poll(
			ID INTEGER PRIMARY KEY ASC,
			UserID INTEGER,
			LastSaved INTEGER,
			CreatedAt INTEGER,
			Type INTEGER,
			Private INTEGER,
			Inactive INTEGER,
			DisplayPercent INTEGER,
			Question TEXT)`); err != nil {
			return err
		}
		if err := sdb.CreateIndex("poll_index ON poll(ID)"); err != nil {
			return err
		}
		if err := sdb.CreateTable(`pollmsg(
			MessageID INTEGER PRIMARY KEY,
			ChatId INTEGER,
			PollID INTEGER)`); err != nil {
			return err
		}
		if err := sdb.CreateIndex("pollmsg_index ON pollmsg(MessageID)"); err != nil {
			return err
		}
		if err := sdb.CreateTable(`pollinlinemsg(
			InlineMessageID TEXT PRIMARY KEY,
			PollID INTEGER)`); err != nil {
			return err
		}
		if err := sdb.CreateTable(`answer(
			ID INTEGER PRIMARY KEY ASC,
			PollID INTEGER,
			OptionID INTEGER,
			LastSaved INTEGER,
			CreatedAt INTEGER,
			UserID INTEGER)`); err != nil {
			return err
		}
		if err := sdb.CreateIndex("answer_index ON answer(PollID)"); err != nil {
			return err
		}
		if err := sdb.CreateTable(`option(
			ID INTEGER PRIMARY KEY ASC,
			PollID INTEGER,
			Ctr INTEGER,
			Text TEXT)`); err != nil {
			return err
		}
		if err := sdb.CreateIndex("option_index ON option(PollID)"); err != nil {
			return err
		}
		if err := sdb.CreateTable(`dialog(
			UserID INTEGER PRIMARY KEY,
			PollID INTEGER,
			state INTEGER)`); err != nil {
			return err
		}
		return sdb.CreateTable(`user(
			ID INTEGER PRIMARY KEY,
			FirstName TEXT,
			LastName Text,
			LastSaved INTEGER,
			CreatedAt INTEGER,
			UserName TEXT)`)
	}},
}
