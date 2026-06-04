package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"account-auto-dispatch/internal/core"
)

const probeJobTimeout = 10 * time.Second
const dispatchEnableSuccessThreshold = 2
const dispatchDisableFailureThreshold = 2

type Server struct {
	mux       *http.ServeMux
	store     core.AccountStore
	scheduler *core.Scheduler
	checker   *core.HealthChecker
	probeJobs *probeJobStore
	control   *probeController
	authToken string
}

func NewServer(store core.AccountStore, scheduler *core.Scheduler, checker *core.HealthChecker) http.Handler {
	s := &Server{mux: http.NewServeMux(), store: store, scheduler: scheduler, checker: checker, probeJobs: newProbeJobStore(), authToken: os.Getenv("AAD_WEB_TOKEN")}
	s.control = newProbeController(s.runProbeBatch)
	if s.control.Snapshot().Enabled {
		s.control.start()
	}
	s.routes()
	go s.reconcileRecoverableAccounts()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.authToken != "" && r.URL.Path == "/login" && r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			writeError(w, http.StatusBadRequest, "invalid login form")
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.FormValue("token")), []byte(s.authToken)) != 1 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(loginHTML))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "aad_token", Value: s.authToken, Path: "/", MaxAge: 30 * 24 * 3600, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if !s.isAuthorized(r) {
		if r.URL.Path == "/" || strings.Contains(r.Header.Get("Accept"), "text/html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(loginHTML))
			return
		}
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) isAuthorized(r *http.Request) bool {
	if s.authToken == "" || r.URL.Path == "/healthz" {
		return true
	}
	candidates := []string{
		r.Header.Get("X-AAD-Token"),
	}
	if cookie, err := r.Cookie("aad_token"); err == nil {
		candidates = append(candidates, cookie.Value)
	}
	for _, candidate := range candidates {
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(s.authToken)) == 1 {
			return true
		}
	}
	return false
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.index)
	s.mux.HandleFunc("GET /healthz", s.healthz)
	s.mux.HandleFunc("POST /v1/accounts", s.createAccount)
	s.mux.HandleFunc("GET /v1/accounts", s.listAccounts)
	s.mux.HandleFunc("GET /v1/dispatch/accounts/", s.getAccountHealth)
	s.mux.HandleFunc("POST /v1/dispatch/select", s.selectAccount)
	s.mux.HandleFunc("POST /v1/dispatch/report-success", s.reportSuccess)
	s.mux.HandleFunc("POST /v1/dispatch/report-failure", s.reportFailure)
	s.mux.HandleFunc("GET /v1/probe-control", s.getProbeControl)
	s.mux.HandleFunc("POST /v1/probe-control", s.updateProbeControl)
	s.mux.HandleFunc("POST /v1/probe-control/run-once", s.runProbeControlOnce)
	s.mux.HandleFunc("POST /v1/dispatch/probe", s.probe)
	s.mux.HandleFunc("GET /v1/dispatch/probe-jobs", s.listProbeJobs)
	s.mux.HandleFunc("GET /v1/dispatch/probe-jobs/", s.getProbeJob)
	s.mux.HandleFunc("POST /v1/accounts/probe-paused", s.setProbePaused)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var req core.Account
	if !decodeJSON(w, r, &req) {
		return
	}
	account, err := s.store.UpsertAccount(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, publicAccount(account))
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.store.ListAccounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, publicAccounts(accounts))
}

func (s *Server) getAccountHealth(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/dispatch/accounts/")
	accountID := strings.TrimSuffix(path, "/health")
	if accountID == "" || accountID == path {
		writeError(w, http.StatusNotFound, "expected /v1/dispatch/accounts/{account_id}/health")
		return
	}
	account, ok, err := s.store.GetAccount(r.Context(), accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	writeJSON(w, http.StatusOK, publicAccount(account))
}

func (s *Server) selectAccount(w http.ResponseWriter, r *http.Request) {
	var req core.SelectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := s.scheduler.Select(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) reportSuccess(w http.ResponseWriter, r *http.Request) {
	var req core.ReportSuccessRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.scheduler.ReportSuccess(r.Context(), req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) reportFailure(w http.ResponseWriter, r *http.Request) {
	var req core.ReportFailureRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.scheduler.ReportFailure(r.Context(), req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) probe(w http.ResponseWriter, r *http.Request) {
	var req core.ProbeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.AccountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}
	job := s.startProbeJob(req.AccountID, req.Model, "manual")
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) startProbeJob(accountID string, model string, trigger string) probeJob {
	job := s.probeJobs.Start(accountID, model, trigger)
	go s.runProbeJob(job.ID, accountID, model, trigger)
	return job
}

func (s *Server) runProbeJob(jobID string, accountID string, model string, trigger string) {
	started := time.Now()
	log.Printf("probe started job_id=%s account_id=%s model=%s trigger=%s", jobID, accountID, model, trigger)
	ctx, cancel := context.WithTimeout(context.Background(), probeJobTimeout)
	defer cancel()

	account, result, ok, err := s.checker.ProbeAccountDryRun(ctx, accountID, model)
	if ok {
		account = s.recordProbeStats(context.Background(), account, result)
	}
	job := s.probeJobs.Finish(jobID, account, result, ok, err)
	log.Printf(
		"probe finished job_id=%s account_id=%s status=%s success=%t error_type=%s trigger=%s duration_ms=%d",
		jobID,
		accountID,
		job.Status,
		result.Success,
		result.ErrorType,
		trigger,
		time.Since(started).Milliseconds(),
	)
}

func (s *Server) recordProbeStats(ctx context.Context, account core.Account, result core.ProbeResult) core.Account {
	now := time.Now().UTC()
	wasDispatchEnabled := account.DispatchEnabled
	account.ProbeTotalCount++
	account.LastProbeAt = &now
	if result.Success {
		account.ProbeSuccessCount++
		account.ConsecutiveSuccesses++
		account.ConsecutiveFailures = 0
		account.ProbeFailureCount = 0
		account.LastSuccessAt = &now
		account.LastErrorType = ""
		account.LastErrorCode = ""
		account.LastErrorMessage = ""
		if shouldEnableRecoveredDispatch(account) {
			enableRecoveredDispatch(&account)
		} else if wasDispatchEnabled {
			account.HealthStatus = core.HealthHealthy
			account.Status = "active"
		}
	} else {
		account.ProbeErrorCount++
		if wasDispatchEnabled {
			account.ConsecutiveSuccesses = 0
		}
		account.ConsecutiveFailures++
		account.ProbeFailureCount++
		account.LastFailedAt = &now
		account.LastErrorType = core.RedactSensitiveText(result.ErrorType)
		if account.LastErrorType == "" {
			account.LastErrorType = core.ErrUnknown
		}
		account.LastErrorCode = core.RedactSensitiveText(result.ErrorCode)
		account.LastErrorMessage = core.RedactSensitiveText(result.ErrorMessage)
		if account.LastErrorMessage == "" {
			account.LastErrorMessage = "probe failed"
		}
		if wasDispatchEnabled && account.ConsecutiveFailures < dispatchDisableFailureThreshold {
			account.HealthStatus = core.HealthSuspect
			account.Status = "active"
			account.DisabledReason = ""
		} else {
			account.DispatchEnabled = false
			account.HealthStatus = core.HealthDisabled
			account.Status = "error"
			account.DisabledReason = account.LastErrorType
			if account.DisabledAt == nil {
				account.DisabledAt = &now
			}
		}
	}
	if err := s.store.UpdateAccount(ctx, account); err != nil {
		log.Printf("probe stats update failed account_id=%s: %v", account.AccountID, err)
	}
	return account
}

func (s *Server) reconcileRecoverableAccounts() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accounts, err := s.store.ListAccounts(ctx)
	if err != nil {
		log.Printf("recoverable account reconciliation list failed: %v", err)
		return
	}
	for _, account := range accounts {
		if !shouldEnableRecoveredDispatch(account) {
			continue
		}
		enableRecoveredDispatch(&account)
		if err := s.store.UpdateAccount(ctx, account); err != nil {
			log.Printf("recoverable account reconciliation update failed account_id=%s: %v", account.AccountID, err)
		}
	}
}

func shouldEnableRecoveredDispatch(account core.Account) bool {
	if account.DispatchEnabled || account.ProbePaused || account.ProbeSuccessCount < dispatchEnableSuccessThreshold {
		return false
	}
	return true
}

func enableRecoveredDispatch(account *core.Account) {
	account.DispatchEnabled = true
	account.HealthStatus = core.HealthHealthy
	account.Status = "active"
	account.DisabledReason = ""
	account.DisabledAt = nil
	account.NextProbeAt = nil
	account.SuspectUntil = nil
	if account.ConsecutiveSuccesses < dispatchEnableSuccessThreshold {
		account.ConsecutiveSuccesses = dispatchEnableSuccessThreshold
	}
}

func (s *Server) runProbeBatch(ctx context.Context, limit int, model string, trigger string) ([]probeJob, error) {
	accounts, err := s.store.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	candidates := s.probeBatchCandidates(accounts)
	if len(candidates) == 0 {
		return []probeJob{}, nil
	}

	jobs := []probeJob{}
	for _, account := range candidates {
		jobs = append(jobs, s.startProbeJob(account.AccountID, model, trigger))
		if len(jobs) >= limit {
			break
		}
	}
	return jobs, nil
}

func (s *Server) probeBatchCandidates(accounts []core.Account) []core.Account {
	activePriority, hasActivePriority := lowestDispatchEnabledPriority(accounts)
	groups := map[int][]core.Account{}
	priorities := []int{}
	seenPriority := map[int]bool{}

	for _, account := range accounts {
		if account.ProbePaused || s.probeJobs.IsAccountRunning(account.AccountID) {
			continue
		}
		priority := normalizedProbePriority(account.Priority)
		if hasActivePriority && priority > activePriority {
			continue
		}
		groups[priority] = append(groups[priority], account)
		if !seenPriority[priority] {
			priorities = append(priorities, priority)
			seenPriority[priority] = true
		}
	}
	sort.Ints(priorities)
	for _, priority := range priorities {
		sort.SliceStable(groups[priority], func(i, j int) bool {
			return probeAccountLess(groups[priority][i], groups[priority][j])
		})
	}
	return roundRobinProbeCandidates(groups, priorities)
}

func lowestDispatchEnabledPriority(accounts []core.Account) (int, bool) {
	best := 0
	found := false
	for _, account := range accounts {
		if account.ProbePaused || !account.DispatchEnabled {
			continue
		}
		priority := normalizedProbePriority(account.Priority)
		if !found || priority < best {
			best = priority
			found = true
		}
	}
	return best, found
}

func roundRobinProbeCandidates(groups map[int][]core.Account, priorities []int) []core.Account {
	candidates := []core.Account{}
	indexes := map[int]int{}
	for {
		added := false
		for _, priority := range priorities {
			index := indexes[priority]
			if index >= len(groups[priority]) {
				continue
			}
			candidates = append(candidates, groups[priority][index])
			indexes[priority] = index + 1
			added = true
		}
		if !added {
			return candidates
		}
	}
}

func normalizedProbePriority(priority int) int {
	if priority <= 0 {
		return int(^uint(0) >> 1)
	}
	return priority
}

func probeAccountLess(left core.Account, right core.Account) bool {
	if left.LastProbeAt == nil && right.LastProbeAt == nil {
		return left.AccountID < right.AccountID
	}
	if left.LastProbeAt == nil {
		return true
	}
	if right.LastProbeAt == nil {
		return false
	}
	return left.LastProbeAt.Before(*right.LastProbeAt)
}

func (s *Server) getProbeJob(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimPrefix(r.URL.Path, "/v1/dispatch/probe-jobs/")
	if jobID == "" {
		writeError(w, http.StatusNotFound, "job id is required")
		return
	}
	job, ok := s.probeJobs.Get(jobID)
	if !ok {
		writeError(w, http.StatusNotFound, "probe job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) listProbeJobs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.probeJobs.List(20))
}

func (s *Server) getProbeControl(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.control.Snapshot())
}

func (s *Server) updateProbeControl(w http.ResponseWriter, r *http.Request) {
	var req probeControlUpdate
	if !decodeJSON(w, r, &req) {
		return
	}
	state := s.control.Update(req)
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) runProbeControlOnce(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.control.RunOnce(r.Context(), "manual_batch")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"jobs": jobs, "count": len(jobs), "control": s.control.Snapshot()})
}

func (s *Server) setProbePaused(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID string `json:"account_id"`
		Paused    bool   `json:"paused"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.AccountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}
	account, ok, err := s.store.GetAccount(r.Context(), req.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	account.ProbePaused = req.Paused
	if err := s.store.UpdateAccount(r.Context(), account); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, publicAccount(account))
}

func publicAccounts(accounts []core.Account) []core.Account {
	out := make([]core.Account, 0, len(accounts))
	for _, account := range accounts {
		out = append(out, publicAccount(account))
	}
	return out
}

func publicAccount(account core.Account) core.Account {
	account.ProbeURL = ""
	account.ProbeBody = ""
	account.ProbeHeaders = nil
	account.CredentialAPIKey = ""
	account.CredentialBaseURL = ""
	account.LastErrorMessage = core.RedactSensitiveText(account.LastErrorMessage)
	return account
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
