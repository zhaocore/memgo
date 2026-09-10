package api

import (
	"net/http"
	"strings"

	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// setupStatus 唯一无鉴权业务端点。
func (s *Server) setupStatus(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.CountUsers(r.Context())
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"needsSetup": n == 0})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	name := validateStringField(&errs, fields, map[string]any{}, "name", true)
	email := validateStringField(&errs, fields, map[string]any{}, "email", true)
	password := validateStringField(&errs, fields, map[string]any{}, "password", true)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	if !validEmail(*email) {
		write422(w, []pyErr{{Type: "value_error", Loc: []any{"body", "email"},
			Msg: "value is not a valid email address", Input: *email}})
		return
	}
	if len(*password) < 8 {
		writeDetail(w, http.StatusBadRequest, "Password must be at least 8 characters.")
		return
	}
	n, err := s.store.CountUsers(r.Context())
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	if n > 0 {
		forbidden(w, "Registration is closed. An admin account already exists.")
		return
	}
	hash, err := auth.HashPassword(*password)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	user := &store.User{ID: newRequestUUID(), Name: *name, Email: *email, PasswordHash: hash, Role: "admin"}
	if err := s.store.CreateUser(r.Context(), user); err != nil {
		// 唯一约束竞态 → registration closed (对齐 IntegrityError 分支)
		forbidden(w, "Registration is closed. An admin account already exists.")
		return
	}
	s.telemetryAdminRegistered(*email)
	s.issueTokens(w, r, user)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	email := validateStringField(&errs, fields, map[string]any{}, "email", true)
	password := validateStringField(&errs, fields, map[string]any{}, "password", true)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	user, err := s.store.GetUserByEmail(r.Context(), *email)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	if user == nil {
		auth.DummyVerify()
		unauthorized(w, "Invalid email or password.")
		return
	}
	if !auth.VerifyPassword(*password, user.PasswordHash) {
		unauthorized(w, "Invalid email or password.")
		return
	}
	if err := s.store.TouchLogin(r.Context(), user.ID); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	s.issueTokens(w, r, user)
}

// issueTokens TokenResponse (access/refresh 双发, jti 落库)。
func (s *Server) issueTokens(w http.ResponseWriter, r *http.Request, user *store.User) {
	access, err := s.tokens.CreateAccessToken(user.ID, user.Role)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	refresh, jti, expires, err := s.tokens.CreateRefreshToken(user.ID)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	if err := s.store.InsertRefreshJTI(r.Context(), jti, user.ID, expires); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access, "refresh_token": refresh, "token_type": "bearer",
	})
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	refreshToken := validateStringField(&errs, fields, map[string]any{}, "refresh_token", true)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	claims, err := s.tokens.Decode(*refreshToken)
	if err != nil {
		unauthorized(w, "Refresh token is no longer valid.")
		return
	}
	if claims.Type != "refresh" {
		unauthorized(w, "Invalid token type.")
		return
	}
	if claims.JTI == "" {
		unauthorized(w, "Refresh token is no longer valid.")
		return
	}
	user, err := s.store.GetUser(r.Context(), claims.Subject)
	if err != nil || user == nil {
		unauthorized(w, "User not found.")
		return
	}
	// 条件 UPDATE: 并发重放恰一成功 (T11)
	okConsumed, err := s.store.ConsumeRefreshJTI(r.Context(), claims.JTI)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	if !okConsumed {
		unauthorized(w, "Refresh token is no longer valid.")
		return
	}
	s.issueTokens(w, r, user)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	writeJSON(w, http.StatusOK, userResponse(user))
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	name := validateStringField(&errs, fields, map[string]any{}, "name", false)
	email := validateStringField(&errs, fields, map[string]any{}, "email", false)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	newName := user.Name
	if name != nil && strings.TrimSpace(*name) != "" {
		newName = strings.TrimSpace(*name)
	}
	if email != nil && *email != user.Email {
		if !validEmail(*email) {
			write422(w, []pyErr{{Type: "value_error", Loc: []any{"body", "email"},
				Msg: "value is not a valid email address", Input: *email}})
			return
		}
		conflict, err := s.store.UpdateUserProfile(r.Context(), user.ID, newName, *email)
		if err != nil {
			writeUpstream(w, requestIDOf(r), err)
			return
		}
		if conflict {
			writeDetail(w, http.StatusConflict, "Email is already in use.")
			return
		}
		_ = *email
	} else if name != nil {
		if _, err := s.store.UpdateUserProfile(r.Context(), user.ID, newName, user.Email); err != nil {
			writeUpstream(w, requestIDOf(r), err)
			return
		}
	}
	updated, err := s.store.GetUser(r.Context(), user.ID)
	if err != nil || updated == nil {
		writeDetail(w, http.StatusNotFound, "User not found.")
		return
	}
	writeJSON(w, http.StatusOK, userResponse(updated))
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	current := validateStringField(&errs, fields, map[string]any{}, "current_password", true)
	newPass := validateStringField(&errs, fields, map[string]any{}, "new_password", true)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	dbUser, err := s.store.GetUser(r.Context(), user.ID)
	if err != nil || dbUser == nil || !auth.VerifyPassword(*current, dbUser.PasswordHash) {
		unauthorized(w, "Current password is incorrect.")
		return
	}
	if len(*newPass) < 8 {
		writeDetail(w, http.StatusBadRequest, "Password must be at least 8 characters.")
		return
	}
	hash, err := auth.HashPassword(*newPass)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	if err := s.store.UpdatePassword(r.Context(), user.ID, hash); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Password updated."})
}

func (s *Server) onboardingComplete(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	useCase := validateStringField(&errs, fields, map[string]any{}, "use_case", true)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	s.telemetryOnboarding(user.Email, *useCase)
	writeJSON(w, http.StatusOK, map[string]any{"message": "Onboarding completed."})
}

// userResponse UserResponse (created_at Z 格式)。
func userResponse(u *store.User) map[string]any {
	return map[string]any{
		"id": u.ID, "name": u.Name, "email": u.Email, "role": u.Role,
		"created_at": store.ZFormat(u.CreatedAt),
	}
}

// validEmail 近似 EmailStr (含 @ 与域点)。
func validEmail(email string) bool {
	at := strings.Index(email, "@")
	return at > 0 && strings.Contains(email[at+1:], ".") && !strings.ContainsAny(email, " \t")
}
