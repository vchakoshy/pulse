package mtproto

import (
	"fmt"
	"time"
)

// Returns user nickname in two formats:
// <id> <First name> @<Username> <Last name> if user has username
// <id> <First name> <Last name> otherwise
func nickname(user TL_user) string {
	if user.Username == "" {
		return fmt.Sprintf("%d %s %s", user.Id, user.First_name, user.Last_name)
	}

	return fmt.Sprintf("%d %s @%s %s", user.Id, user.First_name, user.Username, user.Last_name)
}

// Returns date in RFC822 format
func formatDate(date int32) string {
	unixTime := time.Unix((int64)(date), 0)
	return unixTime.Format(time.RFC822)
}
