package reminddriver

import (
	"github.com/fluffle/sp0rkle/base"
	"github.com/fluffle/sp0rkle/bot"
	"github.com/fluffle/sp0rkle/collections/reminders"
	"github.com/fluffle/sp0rkle/util/datetime"
	"labix.org/v2/mgo/bson"
	"strconv"
	"strings"
	"time"
)

// remind del
func (rd *remindDriver) Del(line *base.Line) {
	list, ok := rd.list[line.Nick]
	if !ok {
		bot.ReplyN(line, "Please use 'remind list' first, " +
			"to be sure of what you're deleting.")
		return
	}
	s := strings.Fields(line.Args[1])
	idx, err := strconv.Atoi(s[len(s)-1])
	if err != nil || idx > len(list) || idx <= 0 {
		bot.ReplyN(line, "Invalid reminder index '%s'", s[len(s)-1])
		return
	}
	idx--
	rd.Forget(list[idx], true)
	delete(rd.list, line.Nick)
	bot.ReplyN(line, "I'll forget that one, then...")
}

// remind list
func (rd *remindDriver) List(line *base.Line) {
	r := rd.RemindersFor(line.Nick)
	c := len(r)
	if c == 0 {
		bot.ReplyN(line, "You have no reminders set.")
		return
	}
	if c > 5 && line.Args[0][0] == '#' {
		bot.ReplyN(line, "You've got lots of reminders, ask me privately.")
		return
	}
	// Save an ordered list of ObjectIds for easy reminder deletion
	bot.ReplyN(line, "You have %d reminders set:", c)
	list := make([]bson.ObjectId, c)
	for i := range r {
		bot.Reply(line, "%d: %s", i+1, r[i].List(line.Nick))
		list[i] = r[i].Id
	}
	rd.list[line.Nick] = list
}

// remind 
func (rd *remindDriver) Set(line *base.Line) {
	// s == remind <target> <reminder> in|at|on <time>
	s := strings.Fields(line.Args[1])
	if len(s) < 5 {
		bot.ReplyN(line, "Invalid remind syntax. Sucka.")
		return
	}
	i := len(s)-1
	for i > 0 {
		lc := strings.ToLower(s[i])
		if lc == "in" || lc == "at" || lc == "on" {
			break
		}
		i--
	}
	if i < 2 {
		bot.ReplyN(line, "Invalid remind syntax. Sucka.")
		return
	}
	reminder := strings.Join(s[2:i], " ")
	timestr := strings.ToLower(strings.Join(s[i+1:], " "))
	// TODO(fluffle): surface better errors from datetime.Parse
	at, ok := datetime.Parse(timestr)
	if !ok {
		bot.ReplyN(line, "Couldn't parse time string '%s'", timestr)
		return
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if at.Before(now) && at.After(start) {
		// Perform some basic hacky corrections before giving up
		if strings.Contains(timestr, "am") || strings.Contains(timestr, "pm") {
			at = at.Add(24 * time.Hour)
		} else {
			at = at.Add(12 * time.Hour)
		}
	}
	if at.Before(now) {
		bot.ReplyN(line, "Time '%s' is in the past.", timestr)
		return
	}
	n, c := line.Storable()
	// TODO(fluffle): Use state tracking! And do this better.
	t := base.Nick(s[1])
	if t.Lower() == strings.ToLower(line.Nick) ||
		t.Lower() == "me" {
		t = n
	}
	r := reminders.NewReminder(reminder, at, t, n, c)
	if err := rd.Insert(r); err != nil {
		bot.ReplyN(line, "Error saving reminder: %v", err)
		return
	}
	// Any previously-generated list of reminders is now obsolete.
	delete(rd.list, line.Nick)
	bot.ReplyN(line, r.Acknowledge())
	rd.Remind(r)
}

// tell
func (rd *remindDriver) Tell(line *base.Line) {
	// s == tell <target> <stuff>
	s := strings.Fields(line.Args[1])
	if len(s) < 3 {
		bot.ReplyN(line, "Tell who what?")
		return
	}
	tell := strings.Join(s[2:], " ")
	n, c := line.Storable()
	t := base.Nick(s[1])
	if t.Lower() == strings.ToLower(line.Nick) ||
		t.Lower() == "me" {
		bot.ReplyN(line, "You're a dick. Oh, wait, that wasn't *quite* it...")
		return
	}
	r := reminders.NewTell(tell, t, n, c)
	if err := rd.Insert(r); err != nil {
		bot.ReplyN(line, "Error saving tell: %v", err)
		return
	}
	// Any previously-generated list of reminders is now obsolete.
	delete(rd.list, line.Nick)
	bot.ReplyN(line, r.Acknowledge())
}