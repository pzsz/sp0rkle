package factdriver

import (
	"fmt"
	"github.com/fluffle/sp0rkle/base"
	"github.com/fluffle/sp0rkle/bot"
	"github.com/fluffle/sp0rkle/collections/factoids"
	"labix.org/v2/mgo/bson"
	"strconv"
	"strings"
	"time"
)

// Factoid chance: 'chance of that is' => sets chance of lastSeen[chan]
func chance(line *base.Line) {
	str := line.Args[1]
	var chance float64

	if strings.HasSuffix(str, "%") {
		// Handle 'chance of that is \d+%'
		if i, err := strconv.Atoi(str[:len(str)-1]); err != nil {
			bot.ReplyN(line, "'%s' didn't look like a % chance to me.", str)
			return
		} else {
			chance = float64(i) / 100
		}
	} else {
		// Assume the chance is a floating point number.
		if c, err := strconv.ParseFloat(str, 64); err != nil {
			bot.ReplyN(line, "'%s' didn't look like a chance to me.", str)
			return
		} else {
			chance = c
		}
	}

	// Make sure the chance we've parsed lies in (0.0,1.0]
	if chance > 1.0 || chance <= 0.0 {
		bot.ReplyN(line, "'%s' was outside possible chance ranges.", str)
		return
	}

	// Retrieve last seen ObjectId, replace with ""
	ls := LastSeen(line.Args[0], "")
	// ok, we're good to update the chance.
	if fact := fc.GetById(ls); fact != nil {
		// Store the old chance, update with the new
		old := fact.Chance
		fact.Chance = chance
		// Update the Modified field
		fact.Modify(line.Storable())
		// And store the new factoid data
		if err := fc.Update(bson.M{"_id": ls}, fact); err == nil {
			bot.ReplyN(line, "'%s' was at %.0f%% chance, now is at %.0f%%.",
				fact.Key, old*100, chance*100)
		} else {
			bot.ReplyN(line, "I failed to replace '%s': %s", fact.Key, err)
		}
	} else {
		bot.ReplyN(line, "Whatever that was, I've already forgotten it.")
	}
}

// Factoid delete: 'forget|delete that' => deletes lastSeen[chan]
func forget(line *base.Line) {
	// Get fresh state on the last seen factoid.
	ls := LastSeen(line.Args[0], "")
	if fact := fc.GetById(ls); fact != nil {
		if err := fc.Remove(bson.M{"_id": ls}); err == nil {
			bot.ReplyN(line, "I forgot that '%s' was '%s'.",
				fact.Key, fact.Value)
		} else {
			bot.ReplyN(line, "I failed to forget '%s': %s", fact.Key, err)
		}
	} else {
		bot.ReplyN(line, "Whatever that was, I've already forgotten it.")
	}
}

// Factoid info: 'fact info key' => some information about key
func info(line *base.Line) {
	key := ToKey(line.Args[1], false)
	count := fc.GetCount(key)
	if count == 0 {
		bot.ReplyN(line, "I don't know anything about '%s'.", key)
		return
	}
	msgs := make([]string, 0, 10)
	if key == "" {
		msgs = append(msgs, fmt.Sprintf("In total, I know %d things.", count))
	} else {
		msgs = append(msgs, fmt.Sprintf("I know %d things about '%s'.",
			count, key))
	}
	if created := fc.GetLast("created", key); created != nil {
		c := created.Created
		msgs = append(msgs, "A factoid")
		if key != "" {
			msgs = append(msgs, fmt.Sprintf("for '%s'", key))
		}
		msgs = append(msgs, fmt.Sprintf("was last created on %s by %s,",
			c.Timestamp.Format(time.ANSIC), c.Nick))
	}
	if modified := fc.GetLast("modified", key); modified != nil {
		m := modified.Modified
		msgs = append(msgs, fmt.Sprintf("modified on %s by %s,",
			m.Timestamp.Format(time.ANSIC), m.Nick))
	}
	if accessed := fc.GetLast("accessed", key); accessed != nil {
		a := accessed.Accessed
		msgs = append(msgs, fmt.Sprintf("and accessed on %s by %s.",
			a.Timestamp.Format(time.ANSIC), a.Nick))
	}
	if info := fc.InfoMR(key); info != nil {
		if key == "" {
			msgs = append(msgs, "These factoids have")
		} else {
			msgs = append(msgs, fmt.Sprintf("'%s' has", key))
		}
		msgs = append(msgs, fmt.Sprintf(
			"been modified %d times and accessed %d times.",
			info.Modified, info.Accessed))
	}
	bot.ReplyN(line, "%s", strings.Join(msgs, " "))
}

// Factoid literal: 'literal key' => info about factoid
func literal(line *base.Line) {
	key := ToKey(line.Args[1], false)
	if count := fc.GetCount(key); count == 0 {
		bot.ReplyN(line, "I don't know anything about '%s'.", key)
		return
	} else if count > 10 && strings.HasPrefix(line.Args[0], "#") {
		bot.ReplyN(line, "I know too much about '%s', ask me privately.", key)
		return
	}

	// Passing an anonymous function to For makes it a little hard to abstract
	// away in lib/factoids. Fortunately this is something of a one-off.
	var fact *factoids.Factoid
	f := func() error {
		if fact != nil {
			bot.ReplyN(line, "[%3.0f%%] %s", fact.Chance*100, fact.Value)
		}
		return nil
	}
	// TODO(fluffle): For() is deprecated nao. FixitFixitFixit.
	if err := fc.Find(bson.M{"key": key}).For(&fact, f); err != nil {
		bot.ReplyN(line, "Something literally went wrong: %s", err)
	}
}

// Factoid replace: 'replace that with' => updates lastSeen[chan]
func replace(line *base.Line) {
	ls := LastSeen(line.Args[0], "")
	if fact := fc.GetById(ls); fact != nil {
		// Store the old factoid value
		old := fact.Value
		// Replace the value with the new one
		fact.Value = line.Args[1]
		// Update the Modified field
		fact.Modify(line.Storable())
		// And store the new factoid data
		if err := fc.Update(bson.M{"_id": ls}, fact); err == nil {
			bot.ReplyN(line, "'%s' was '%s', now is '%s'.",
				fact.Key, old, fact.Value)
		} else {
			bot.ReplyN(line, "I failed to replace '%s': %s", fact.Key, err)
		}
	} else {
		bot.ReplyN(line, "Whatever that was, I've already forgotten it.")
	}
}

// Factoid search: 'fact search regexp' => list of possible key matches
func search(line *base.Line) {
	keys := fc.GetKeysMatching(line.Args[1])
	if keys == nil || len(keys) == 0 {
		bot.ReplyN(line, "I couldn't think of anything matching '%s'.",
			line.Args[1])
		return
	}
	// RESULTS.
	count := len(keys)
	if count > 10 {
		res := strings.Join(keys[:10], "', '")
		bot.ReplyN(line,
			"I found %d keys matching '%s', here's the first 10: '%s'.",
			count, line.Args[1], res)
	} else {
		res := strings.Join(keys, "', '")
		bot.ReplyN(line,
			"%s: I found %d keys matching '%s', here they are: '%s'.",
			count, line.Args[1], res)
	}
}
