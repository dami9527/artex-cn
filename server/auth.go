package server

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	jwtKeyFilename = "jwt.key"
	authPassKey    = "auth.password_hash"
	jwtTTL         = 7 * 24 * time.Hour
	keyChars       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// 하한은 setup 페이지의 프런트 검증과 맞춘 값이다. 검증을 프런트에만 두면 API 를
	// 직접 호출해 우회할 수 있다. 상한은 bcrypt 의 한계로, 72 바이트를 넘으면
	// GenerateFromPassword 가 ErrPasswordTooLong 을 반환하므로 뜻이 불분명한
	// 「비밀번호 암호화 실패」를 보여주기 전에 미리 막는다.
	minPasswordRunes = 8
	maxPasswordBytes = 72
)

// 인증 엔드포인트가 HTTP 응답으로 돌려주는 사용자 노출 문구다. 한국어 UI 에서 로그인·
// 비밀번호 설정이 실패하면 이 문구가 그대로 토스트로 뜨므로 한국어로 둔다. 자격 증명
// 오류 문구는 로그인 화면(web messages auth.login.errorCredential)과 표기를 맞췄다.
// token 은 기술 용어라 원문 그대로 둔다(로그·주석은 BRIEF 방침상 최하위라 손대지 않음).
const (
	authErrUnauthorized         = "인증이 필요합니다"
	authErrTokenInvalid         = "token 이 유효하지 않거나 만료되었습니다"
	authErrPasswordAlreadySet   = "비밀번호가 이미 설정되어 있습니다"
	authErrPasswordEmpty        = "비밀번호를 입력해 주세요"
	authErrNewPasswordEmpty     = "새 비밀번호를 입력해 주세요"
	authErrPasswordHash         = "비밀번호 암호화에 실패했습니다"
	authErrSaveFailedPrefix     = "저장에 실패했습니다: "
	authErrTokenGen             = "token 생성에 실패했습니다"
	authErrBadRequest           = "요청 형식이 올바르지 않습니다"
	authErrPasswordNotInit      = "비밀번호가 초기화되지 않았습니다. 먼저 비밀번호를 설정해 주세요"
	authErrCurrentPasswordWrong = "현재 비밀번호가 올바르지 않습니다"
	authErrBadCredential        = "사용자 이름 또는 비밀번호가 올바르지 않습니다"

	// authErrDataSourceUnavailable 은 비밀번호 관련 읽기 작업이 실패했을 때 돌려주는
	// 공통 문구다. 이 핸들러들은 "읽지 못함"을 "설정되지 않음"으로 취급하면 안 된다.
	// 실제로 authInit 이 그렇게 동작해, 데이터베이스 오류 시 미인증 요청이 기존 관리자
	// 비밀번호를 덮어쓸 수 있었다(상류 a951e4a 에서 수정).
	authErrDataSourceUnavailable = "데이터 소스를 일시적으로 사용할 수 없습니다. 잠시 후 다시 시도해 주세요"
)

// validatePassword 는 통과하면 빈 문자열을, 아니면 사용자에게 그대로 보여줄 한국어
// 사유를 돌려준다. 하한은 setup 페이지의 프런트 검증과 맞췄지만, 검증을 프런트에만
// 두면 API 를 직접 호출해 우회할 수 있으므로 서버에서도 강제한다. 상한은 bcrypt 의
// 한계다(72 바이트 초과 시 GenerateFromPassword 가 ErrPasswordTooLong 을 반환하므로,
// 뜻이 불분명한 「비밀번호 암호화 실패」를 보여주기 전에 미리 막는다).
func validatePassword(pw string) string {
	if utf8.RuneCountInString(pw) < minPasswordRunes {
		return fmt.Sprintf("비밀번호는 최소 %d자 이상이어야 합니다", minPasswordRunes)
	}
	if len(pw) > maxPasswordBytes {
		return fmt.Sprintf("비밀번호는 %d바이트를 넘을 수 없습니다", maxPasswordBytes)
	}
	return ""
}

// loadOrCreateJWTKey reads the 32-byte signing key from keyDir/jwt.key. keyDir is
// the project base dir (next to the executable), NOT the browsable workspace root
// (dataDir) — the signing key must never be listable/downloadable via the file
// manager. Legacy installs kept it at dataDir/jwt.key; if present there and not yet
// at the new location, it is migrated (key preserved, so sessions stay valid) and
// the old file removed so it disappears from the workspace. On first run a random
// key is generated and persisted.
func loadOrCreateJWTKey(keyDir, dataDir string) ([]byte, error) {
	path := filepath.Join(keyDir, jwtKeyFilename)
	// one-time migration out of the old in-workspace location.
	if legacy := filepath.Join(dataDir, jwtKeyFilename); legacy != path {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if data, rerr := os.ReadFile(legacy); rerr == nil {
				if werr := os.WriteFile(path, data, 0o600); werr == nil {
					_ = os.Remove(legacy)
					log.Printf("[auth] JWT 키를 %s 에서 %s 로 이전(탐색 가능한 워크스페이스 밖으로 이동)", legacy, path)
				}
			}
		}
	}
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) >= 32 {
		return []byte(strings.TrimSpace(string(data))), nil
	}
	buf := make([]byte, 32)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(keyChars))))
		if err != nil {
			return nil, fmt.Errorf("generate jwt key: %w", err)
		}
		buf[i] = keyChars[n.Int64()]
	}
	if err := os.WriteFile(path, buf, 0600); err != nil {
		return nil, fmt.Errorf("write jwt key: %w", err)
	}
	log.Printf("[auth] 새 JWT 키를 %s 에 기록", path)
	return buf, nil
}

// signJWT issues a 7-day HS256 token for user ARTEX.
func signJWT(key []byte) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "ARTEX",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}).SignedString(key)
}

// verifyJWT returns true when tokenStr is a valid, non-expired HS256 token.
func verifyJWT(tokenStr string, key []byte) bool {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	})
	return err == nil && t.Valid
}

// extractToken reads the JWT from Authorization: Bearer header,
// artex_token cookie, or ?token= query param (for SSE connections).
func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if c, err := r.Cookie("artex_token"); err == nil && c.Value != "" {
		return c.Value
	}
	return r.URL.Query().Get("token")
}

// requireAuth wraps h with JWT validation.
// /api/auth/* and /api/health are exempt.
func (s *Server) requireAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/auth/") || p == "/api/health" {
			h.ServeHTTP(w, r)
			return
		}
		tok := extractToken(r)
		if tok == "" {
			writeErr(w, 401, authErrUnauthorized)
			return
		}
		if !verifyJWT(tok, s.jwtKey) {
			writeErr(w, 401, authErrTokenInvalid)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// GET /api/auth/status — reports whether the admin password has been initialised.
// 읽기 실패는 initialized:false 가 아니라 503 으로 돌려줘야 한다. 프런트는
// initialized:false 를 받으면 사용자를 /setup 으로 보내 비밀번호를 설정하게 하는데
// (login/page.tsx), 데이터베이스 장애를 200 으로 포장하면 사용자를 기존 비밀번호를
// 덮어쓰는 경로로 밀어 넣는 셈이 된다.
func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	hash, _, err := pg.GetSetting(authPassKey)
	if err != nil {
		writeErr(w, 503, authErrDataSourceUnavailable)
		return
	}
	writeJSON(w, 200, map[string]any{"initialized": hash != ""})
}

// POST /api/auth/init — sets the password for the first time; rejected if already set.
func (s *Server) authInit(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	existing, _, err := pg.GetSetting(authPassKey)
	if err != nil {
		writeErr(w, 503, authErrDataSourceUnavailable)
		return
	}
	if existing != "" {
		writeErr(w, 403, authErrPasswordAlreadySet)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil || req.Password == "" {
		writeErr(w, 400, authErrPasswordEmpty)
		return
	}
	if msg := validatePassword(req.Password); msg != "" {
		writeErr(w, 400, msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, 500, authErrPasswordHash)
		return
	}
	// INSERT ... ON CONFLICT DO NOTHING 을 쓰고 upsert 를 쓰지 않는다. 위의 GetSetting 은
	// 빠른 실패 경로일 뿐이고, "최초 1회만 설정 가능"이라는 보장은 기본 키 제약이 진다.
	// bcrypt 는 수십 밀리초가 걸리므로 그 사이 다른 요청이 먼저 비밀번호를 설정할 수 있고,
	// 읽기 검사 자체도 장애로 무효가 될 수 있다.
	inserted, err := pg.InsertSettingIfAbsent(authPassKey, string(hash))
	if err != nil {
		writeErr(w, 500, authErrSaveFailedPrefix+err.Error())
		return
	}
	if !inserted {
		writeErr(w, 403, authErrPasswordAlreadySet)
		return
	}
	tok, err := signJWT(s.jwtKey)
	if err != nil {
		writeErr(w, 500, authErrTokenGen)
		return
	}
	writeJSON(w, 200, map[string]any{"token": tok})
}

// POST /api/auth/change-password — changes the admin password. Requires a valid
// token (this route is under /api/auth/* which requireAuth exempts, so the token
// is validated here) AND the current password.
func (s *Server) authChangePassword(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	if !verifyJWT(extractToken(r), s.jwtKey) {
		writeErr(w, 401, authErrUnauthorized)
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, authErrBadRequest)
		return
	}
	if req.NewPassword == "" {
		writeErr(w, 400, authErrNewPasswordEmpty)
		return
	}
	if msg := validatePassword(req.NewPassword); msg != "" {
		writeErr(w, 400, msg)
		return
	}
	hash, ok, err := pg.GetSetting(authPassKey)
	if err != nil {
		writeErr(w, 503, authErrDataSourceUnavailable)
		return
	}
	if !ok || hash == "" {
		writeErr(w, 403, authErrPasswordNotInit)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.OldPassword)); err != nil {
		writeErr(w, 401, authErrCurrentPasswordWrong)
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, 500, authErrPasswordHash)
		return
	}
	if err := pg.SetSetting(authPassKey, string(newHash)); err != nil {
		writeErr(w, 500, authErrSaveFailedPrefix+err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/auth/login — validates username/password and returns a JWT.
func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, authErrBadRequest)
		return
	}
	if req.Username != "ARTEX" {
		writeErr(w, 401, authErrBadCredential)
		return
	}
	hash, ok, err := pg.GetSetting(authPassKey)
	if err != nil {
		writeErr(w, 503, authErrDataSourceUnavailable)
		return
	}
	if !ok || hash == "" {
		writeErr(w, 403, authErrPasswordNotInit)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		writeErr(w, 401, authErrBadCredential)
		return
	}
	tok, err := signJWT(s.jwtKey)
	if err != nil {
		writeErr(w, 500, authErrTokenGen)
		return
	}
	writeJSON(w, 200, map[string]any{"token": tok})
}
