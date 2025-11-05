package main

import (
	"context"
	"fmt"
	"strings"

	ircevents "github.com/Jamie-38/stream-pipeline/internal/irc_events"
	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func Classify_line(ctx context.Context, readerCh <-chan string, parseCh chan<- ircevents.Event, membershipCh chan<- types.MembershipEvent, username string) {
	for {
		select {
		case <-ctx.Done():
			return

		case line := <-readerCh:
			if len(line) == 0 {
				continue
			}
			i := 0

			// ---- TAGS (optional, starts with '@') ----
			var tags string
			var tagsMap map[string]string
			if i < len(line) && line[i] == '@' {
				j := strings.IndexByte(line[i:], ' ')
				if j < 0 {
					fmt.Println("skip: malformed tags (no space)")
					continue
				}
				tags = line[i+1 : i+j] // drop '@'
				tagsMap = parseTags(tags)
				i += j + 1
			} else {
				tagsMap = map[string]string{}
			}

			// ---- PREFIX (optional, starts with ':') ----
			var prefix string
			if i < len(line) && line[i] == ':' {
				j := strings.IndexByte(line[i:], ' ')
				if j < 0 {
					fmt.Println("skip: malformed prefix (no space)")
					continue
				}
				prefix = line[i+1 : i+j] // drop ':'
				i += j + 1
			}

			// ---- COMMAND (required) ----
			if i >= len(line) {
				fmt.Println("skip: missing command")
				continue
			}
			var command string
			if j := strings.IndexByte(line[i:], ' '); j < 0 {
				command = line[i:]
				i = len(line)
			} else {
				command = line[i : i+j]
				i += j + 1
			}

			// ---- PARAMS / TRAILING (optional) ----
			var params []string
			var trailing string
			if i <= len(line) {
				if k := strings.Index(line[i:], " :"); k >= 0 {
					paramsPart := line[i : i+k]
					trailing = line[i+k+2:] // everything to end (may contain spaces)
					params = fieldsNoEmpty(paramsPart)
				} else {
					params = fieldsNoEmpty(line[i:])
					trailing = ""
				}
			}

			// ---- DEBUG: show core tokens ----
			// fmt.Println("tags   :", tags)
			fmt.Println("prefix :", prefix)
			fmt.Println("command:", command)
			fmt.Println("params :", params)
			fmt.Println("trail  :", trailing)

			// ---- COMMAND HANDLERS (minimal) ----
			switch command {
			case "PRIVMSG":
				if len(params) == 0 || len(trailing) == 0 {
					fmt.Println("skip: malformed PRIVMSG (missing channel or text)")
					continue
				}

				// Prefer authoritative IDs from tags (fall back to names if missing)
				// Assume you turned the raw `tags` string into tagsMap via a tiny helper.
				userID := tagsMap["user-id"]
				channelID := tagsMap["room-id"]

				// Fallbacks (don’t crash if tags are missing during early testing)
				if userID == "" {
					userID = loginFromPrefix(prefix) // login, not ideal, but okay as a fallback
				}
				chanLogin := params[0] // e.g. "#vedal987"
				if channelID == "" {
					channelID = strings.TrimPrefix(chanLogin, "#")
				}

				evt := ircevents.PrivMsg{
					UserID:    userID,
					ChannelID: channelID,
					Text:      trailing, // full message text
				}

				select {
				case parseCh <- evt: // PrivMsg satisfies Event; implicit interface conversion
				case <-ctx.Done():
					return
				}

			case "JOIN", "PART":
				if len(params) == 0 {
					fmt.Println("skip: malformed", command, "(missing channel)")
					continue
				}

				userLogin := strings.ToLower(loginFromPrefix(prefix))
				if userLogin == "" {
					// no prefix → can’t attribute; ignore
					continue
				}

				// own JOIN/PART as membership confirmations
				if userLogin != username {
					continue
				}

				ch := strings.ToLower(params[0])
				if !strings.HasPrefix(ch, "#") {
					ch = "#" + ch
				}

				// emit membership signal
				evt := types.MembershipEvent{
					Op:      command, // "JOIN" or "PART"
					Channel: ch,
				}
				select {
				case membershipCh <- evt:
				case <-ctx.Done():
					return
				default:
					// drop if full; rectifier will reconcile on next tick/timeout
				}

			default:
				// USERNOTICE, ROOMSTATE, numerics, etc
			}
		}
	}
}

func fieldsNoEmpty(s string) []string {
	parts := strings.Fields(s)
	// strings.Fields already drops empties; keep as a separate helper for clarity/extensibility.
	return parts
}

func loginFromPrefix(prefix string) string {
	// prefix looks like "login!login@login.tmi.twitch.tv"
	if prefix == "" {
		return ""
	}
	if idx := strings.IndexByte(prefix, '!'); idx >= 0 {
		return prefix[:idx]
	}
	// Sometimes prefix can be just "tmi.twitch.tv" for numerics; return as-is.
	return prefix
}

// tagsStr is everything after '@' up to the first space.
// Example: "badge-info=subscriber/10;badges=subscriber/9;color=#00FF7F;user-id=464309918"
func parseTags(tagsStr string) map[string]string {
	tags := make(map[string]string, 16)
	if tagsStr == "" {
		return tags
	}
	for _, pair := range strings.Split(tagsStr, ";") {
		if pair == "" {
			continue
		}
		if eq := strings.IndexByte(pair, '='); eq >= 0 {
			k := pair[:eq]
			v := pair[eq+1:]
			tags[k] = unescapeIRCv3(v)
		} else {
			// "flagOnly" tags are allowed by spec; store as "1"
			tags[pair] = "1"
		}
	}
	return tags
}

// IRCv3 tag value escapes: \s (space), \: (:), \; (;), \\ (\), \r, \n
func unescapeIRCv3(s string) string {
	// Fast path: nothing to unescape
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 's':
			b.WriteByte(' ')
		case ':':
			b.WriteByte(':')
		case ';':
			b.WriteByte(';')
		case '\\':
			b.WriteByte('\\')
		case 'r':
			b.WriteByte('\r')
		case 'n':
			b.WriteByte('\n')
		default:
			// Unknown escape: keep as-is (spec is small; this is safe)
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
