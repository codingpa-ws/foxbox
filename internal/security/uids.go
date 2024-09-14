package security

import (
	"fmt"
	"os/user"
	"strconv"
)

func GetUserIdentifiers() (uid, gid int, err error) {
	user, err := user.Current()
	if err != nil {
		return 0, 0, fmt.Errorf("getting current user: %w", err)
	}

	uid, err = strconv.Atoi(user.Uid)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing user uid (%s): %w", user.Uid, err)
	}
	gid, err = strconv.Atoi(user.Gid)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing user gid (%s): %w", user.Gid, err)
	}

	return
}
