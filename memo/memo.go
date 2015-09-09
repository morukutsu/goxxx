// The MIT License (MIT)
//
// Copyright (c) 2015 Arnaud Vazard
//
// See LICENSE file.

// Package memo allow to leave memo for AFK users
package memo

import (
	"database/sql"
	"fmt"
	"github.com/emirozer/go-helpers"
	"github.com/thoj/go-ircevent"
	"github.com/vaz-ar/goxxx/core"
	"log"
	"strings"
)

// Help messages
const (
	HelpMemo     = "\t!memo/!m <nick> <message> \t=> Leave a memo for another user"                               // Help message for the memo command
	HelpMemostat = "\t!memostat/!ms \t\t\t\t\t=> Get the list of the unread memos (List only the memos you left)" // Help message for the memo status command
)

var (
	memoCmd     = []string{"!memo", "!m"}      // Slice containing the possible memo commands
	memostatCmd = []string{"!memostat", "!ms"} // Slice containing the possible memo status commands
	dbPtr       *sql.DB                        // Database pointer
)

// data stores memo informations, based on the database table "Memo".
type data struct {
	id       int
	date     string
	message  string
	userFrom string
	userTo   string
}

// Init stores the database pointer and initialises the database table "Memo" if necessary.
func Init(db *sql.DB) {
	dbPtr = db
	sqlStmt := `CREATE TABLE IF NOT EXISTS Memo (
    id integer NOT NULL PRIMARY KEY,
    user_to TEXT,
    user_from TEXT,
    message TEXT,
    date DATETIME DEFAULT CURRENT_TIMESTAMP);`

	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}
}

// HandleMemoCmd handles memo commands.
func HandleMemoCmd(event *irc.Event, callback func(*core.ReplyCallbackData)) bool {
	fields := strings.Fields(event.Message())
	// fields[0]  => Command
	// fields[1]  => recipient's nick
	// fields[2:] => message
	if len(fields) < 3 || !helpers.StringInSlice(fields[0], memoCmd) {
		return false
	}
	memo := data{
		userTo:   fields[1],
		userFrom: event.Nick,
		message:  strings.Join(fields[2:], " ")}

	sqlStmt := "INSERT INTO Memo (user_to, user_from, message) VALUES ($1, $2, $3)"
	_, err := dbPtr.Exec(sqlStmt, memo.userTo, memo.userFrom, memo.message)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}

	if callback != nil {
		callback(&core.ReplyCallbackData{
			Message: fmt.Sprintf("%s: memo for %s saved", memo.userFrom, memo.userTo),
			Nick:    memo.userFrom})
	}
	return true
}

// SendMemo is a message handler that will send memo(s) to an user when he post a message for the first time after a memo for him was created.
func SendMemo(event *irc.Event, callback func(*core.ReplyCallbackData)) {
	user := event.Nick
	sqlQuery := "SELECT id, user_from, message, strftime('%d/%m/%Y @ %H:%M', datetime(date, 'localtime')) FROM Memo WHERE user_to = $1;"
	rows, err := dbPtr.Query(sqlQuery, user)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlQuery)
	}
	defer rows.Close()

	userTo := event.Nick
	var memoList []data
	for rows.Next() {
		var memo data
		rows.Scan(&memo.id, &memo.userFrom, &memo.message, &memo.date)
		memoList = append(memoList, memo)
		callback(&core.ReplyCallbackData{
			Message: fmt.Sprintf("%s: memo from %s => \"%s\" (%s)", userTo, memo.userFrom, memo.message, memo.date),
			Nick:    userTo})
	}
	rows.Close()

	for _, memo := range memoList {
		sqlQuery = "DELETE FROM Memo WHERE id = $1"
		_, err = dbPtr.Exec(sqlQuery, memo.id)
		if err != nil {
			log.Fatalf("%q: %s\n", err, sqlQuery)
		}
	}
}

// HandleMemoStatusCmd handles memo status commands.
func HandleMemoStatusCmd(event *irc.Event, callback func(*core.ReplyCallbackData)) bool {
	fields := strings.Fields(event.Message())
	// fields[0]  => Command
	if len(fields) == 0 || !helpers.StringInSlice(fields[0], memostatCmd) {
		return false
	}

	sqlQuery := "SELECT id, user_to, message, strftime('%d/%m/%Y @ %H:%M', datetime(date, 'localtime')) FROM Memo WHERE user_from = $1 ORDER BY id"
	rows, err := dbPtr.Query(sqlQuery, event.Nick)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlQuery)
	}
	defer rows.Close()

	var memo data
	for rows.Next() {
		rows.Scan(&memo.id, &memo.userTo, &memo.message, &memo.date)
		callback(&core.ReplyCallbackData{
			Message: fmt.Sprintf("Memo for %s: \"%s\" (%s)", memo.userTo, memo.message, memo.date),
			Nick:    event.Nick})
	}
	rows.Close()

	if memo.id == 0 {
		callback(&core.ReplyCallbackData{Message: "No memo saved", Nick: event.Nick})
	}
	return true
}
