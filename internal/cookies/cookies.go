package cookies

import (
	"net/http"
	"os"
)

const (
	AccessTokenCookieName  = "auth_token"
	RefreshTokenCookieName = "refresh_token"
)

func SetAccessTokenCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(15 * 60),
	})
}

func SetRefreshTokenCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(7 * 24 * 60 * 60),
	})
}

func ClearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func GetRefreshTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// secureCookies is an operator-controlled deployment flag (SECURE_COOKIES=1|true).
// It is intentionally NOT derived from X-Forwarded-Proto, which a client can
// spoof: a request over plain HTTP could otherwise force an insecure (HTTP)
// cookie and expose tokens to network sniffing (CONCERNS.md #12). Set it when
// the app is served over HTTPS (typically behind a TLS-terminating proxy).
var secureCookies = os.Getenv("SECURE_COOKIES") == "1" || os.Getenv("SECURE_COOKIES") == "true"

func IsSecureRequest(r *http.Request) bool {
	return r.TLS != nil || secureCookies
}
