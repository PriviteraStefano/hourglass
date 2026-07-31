package auth

import "errors"

// ErrTokenReuse is returned by Service.Refresh when a refresh token that was
// already rotated or revoked is presented again. The repository detects the
// replay (via ports.ErrTokenReuse) and revokes the entire token family; the
// handler maps this to 401 with both auth cookies cleared.
var ErrTokenReuse = errors.New("refresh token reuse detected")
