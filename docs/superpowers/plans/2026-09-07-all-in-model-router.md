# All-In Model Router Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Один аккаунт `All-In`, в чьём `/model` видны модели всех Claude-логинов и всех subscription-профилей, с переключением между ними без перезапуска панели.

**Architecture:** Профиль All-In пишет в свой settings-JSON ключ `modelPicker` со строками вида `wisp/<source>/<model>`. Запуск идёт через обёртку `wisp-deck-tui claude-allin`, которая поднимает loopback-роутер и переписывает `ANTHROPIC_BASE_URL` в overlay'е (в точности как уже делает `claude-rolefix`). Роутер разбирает `model`, подменяет креды и форвардит на нужный апстрим.

**Tech Stack:** Go (`internal/allin`, cobra-команда в `cmd/wisp-deck-tui`), bash (`lib/tmux-session.sh`), macOS Keychain через `security`.

**Spec:** `docs/all-in-model-router.md`

## Global Constraints

- Грамматика id: `wisp/<source>/<model>`; `<source>` — ровно один сегмент, `<model>` может содержать слэши. Режем только по первым двум сегментам.
- `<source>` = `acct.<dir>` (Claude-логин) либо `cfg.<file>` (профиль, имя файла без `.json`). `acct.default` — Keychain-логин без `CLAUDE_CONFIG_DIR`.
- Keychain-сервис: `Claude Code-credentials-<sha256(configDir)[:8]>`; для Default — `Claude Code-credentials` без суффикса.
- Суффикс `[1m]` срезается перед форвардом, и тогда в `Anthropic-Beta` добавляется `context-1m-2025-08-07`.
- **Протухший токен отдаётся как HTTP 400, а не 401.** Claude Code повторяет 401 около 11 раз; детерминированная ошибка обязана быть 400 (см. память `claude-code-retries-401-and-429`).
- Роутер обслуживает и `/v1/messages`, и `/v1/messages/have-to-count` → фактически любой путь: проксируем всё, `/v1/messages/count_tokens` даёт 156 запросов на один `/context`.
- Стриминг без буферизации: `FlushInterval: -1`. Буферизация убивает keep-alive и будит watchdog.
- Ничто здесь не имеет права стоить пользователю сессию: нечитаемый overlay, отсутствующий эндпоинт или уже локальный эндпоинт → запускаем ребёнка ровно так, как он запустился бы сам.
- В список не попадают модели с окном меньше 200000 токенов.
- Каждый `exec.Command` в продакшн-Go обязан быть зарегистрирован в `auditedProductionProcessCalls` (`test/bash/idle_sound_ownership_test.go`), иначе сборка красная.
- Тесты: только по изменённым файлам. `shellcheck` — только по изменённым скриптам.

---

### Task 1: Грамматика id и маршрутизация

**Files:**
- Create: `internal/allin/route.go`
- Test: `internal/allin/route_test.go`

**Interfaces:**
- Consumes: ничего.
- Produces: `type Kind int` с константами `KindSession`, `KindAccount`, `KindConfig`; `type Target struct { Kind Kind; Source string; Model string; Want1M bool }`; `func Route(model string) Target`.

- [ ] **Step 1: Write the failing test**

```go
package allin

import "testing"

func TestRoute_leaves_a_plain_model_on_the_session(t *testing.T) {
	got := Route("claude-opus-5")
	if got.Kind != KindSession || got.Model != "claude-opus-5" || got.Want1M {
		t.Fatalf("got %+v", got)
	}
}

func TestRoute_splits_an_account_row(t *testing.T) {
	got := Route("wisp/acct.personal/claude-opus-5")
	if got.Kind != KindAccount || got.Source != "personal" || got.Model != "claude-opus-5" {
		t.Fatalf("got %+v", got)
	}
}

func TestRoute_keeps_slashes_inside_the_model_id(t *testing.T) {
	got := Route("wisp/cfg.featherless/TurboVadim/Qwen3.8-27B-OBLITERATED")
	if got.Kind != KindConfig || got.Source != "featherless" {
		t.Fatalf("got %+v", got)
	}
	if got.Model != "TurboVadim/Qwen3.8-27B-OBLITERATED" {
		t.Fatalf("model %q", got.Model)
	}
}

func TestRoute_strips_the_1m_marker_and_reports_it(t *testing.T) {
	got := Route("wisp/acct.default/claude-opus-5[1m]")
	if got.Model != "claude-opus-5" || !got.Want1M {
		t.Fatalf("got %+v", got)
	}
}

func TestRoute_treats_an_unparseable_prefix_as_the_session(t *testing.T) {
	for _, id := range []string{"wisp/", "wisp/onlysource", "wisp/bogus.x/m"} {
		if got := Route(id); got.Kind != KindSession {
			t.Fatalf("%q routed to %+v", id, got)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/allin/ -run TestRoute -v`
Expected: FAIL — пакет не существует / `undefined: Route`.

- [ ] **Step 3: Write minimal implementation**

```go
// Package allin routes one Claude Code session across every configured
// subscription: each picker row names its source in the model id, and the
// loopback router swaps the credential for that source.
package allin

import "strings"

// Kind says which credential a picker row asks for.
type Kind int

const (
	// KindSession leaves the session's own credential and endpoint alone.
	KindSession Kind = iota
	KindAccount
	KindConfig
)

// Target is one parsed picker row.
type Target struct {
	Kind   Kind
	Source string
	Model  string
	Want1M bool
}

const (
	rowPrefix     = "wisp/"
	accountPrefix = "acct."
	configPrefix  = "cfg."
)

// Route parses a picker row. A row this build does not recognise is the
// session's own, never an error: the turn must still run.
func Route(model string) Target {
	trimmed, want1m := strip1M(model)
	rest, ok := strings.CutPrefix(trimmed, rowPrefix)
	if !ok {
		return Target{Kind: KindSession, Model: trimmed, Want1M: want1m}
	}
	source, id, ok := strings.Cut(rest, "/")
	if !ok || source == "" || id == "" {
		return Target{Kind: KindSession, Model: trimmed, Want1M: want1m}
	}
	switch {
	case strings.HasPrefix(source, accountPrefix):
		return Target{Kind: KindAccount, Source: source[len(accountPrefix):], Model: id, Want1M: want1m}
	case strings.HasPrefix(source, configPrefix):
		return Target{Kind: KindConfig, Source: source[len(configPrefix):], Model: id, Want1M: want1m}
	}
	return Target{Kind: KindSession, Model: trimmed, Want1M: want1m}
}

// strip1M removes the marker Claude Code reads off the raw model string to
// grant a 1M window. The upstream never sees it; the beta header carries it.
func strip1M(model string) (string, bool) {
	if len(model) >= 4 && strings.EqualFold(model[len(model)-4:], "[1m]") {
		return model[:len(model)-4], true
	}
	return model, false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/allin/ -run TestRoute -v`
Expected: PASS, все пять тестов.

- [ ] **Step 5: Commit**

```bash
git add internal/allin/route.go internal/allin/route_test.go
git commit -m "feat(allin): parse a picker row into its source, model and 1M marker"
```

---

### Task 2: Ростер строк пикера

**Files:**
- Create: `internal/allin/roster.go`
- Test: `internal/allin/roster_test.go`

**Interfaces:**
- Consumes: `claudeconfig.Load`, `claudeconfig.ProviderForConfig`, `claudeconfig.ModelsForConfig`, `claudeconfig.ReadCustomModel`, `claudeconfig.ConfigReady` из `internal/claudeconfig`.
- Produces: `type Env struct { AccountsList, AccountsDir, ConfigsList, ConfigsDir string }`; `type Row struct { Model, Label, Description, BehavesAs string }` с JSON-тегами `model,label,description,behavesAs`; `func Roster(env Env) []Row`.

- [ ] **Step 1: Write the failing test**

```go
package allin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rosterEnv(t *testing.T) Env {
	t.Helper()
	dir := t.TempDir()
	accounts := filepath.Join(dir, "claude-accounts")
	configs := filepath.Join(dir, "claude-configs")
	if err := os.MkdirAll(filepath.Join(accounts, "personal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(dir, "claude-accounts.list"), "Personal:personal\n")
	write(filepath.Join(dir, "claude-configs.list"), "Zhipu GLM:zhipu-glm.json\n")
	write(filepath.Join(configs, "zhipu-glm.json"),
		`{"env":{"ANTHROPIC_BASE_URL":"https://api.z.ai/api/anthropic","ANTHROPIC_AUTH_TOKEN":"k"}}`)
	return Env{
		AccountsList: filepath.Join(dir, "claude-accounts.list"),
		AccountsDir:  accounts,
		ConfigsList:  filepath.Join(dir, "claude-configs.list"),
		ConfigsDir:   configs,
	}
}

func models(rows []Row) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Model)
	}
	return out
}

func has(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func TestRoster_lists_the_default_login_and_every_registered_account(t *testing.T) {
	got := models(Roster(rosterEnv(t)))
	if !has(got, "wisp/acct.default/claude-opus-5") {
		t.Fatalf("no default opus row in %v", got)
	}
	if !has(got, "wisp/acct.personal/claude-opus-5") {
		t.Fatalf("no personal opus row in %v", got)
	}
}

func TestRoster_marks_a_1M_capable_model_so_the_client_grants_the_window(t *testing.T) {
	got := models(Roster(rosterEnv(t)))
	if !has(got, "wisp/acct.default/claude-opus-5[1m]") {
		t.Fatalf("no 1m opus row in %v", got)
	}
}

func TestRoster_lists_a_configured_providers_models(t *testing.T) {
	got := models(Roster(rosterEnv(t)))
	if !has(got, "wisp/cfg.zhipu-glm/glm-4.7") {
		t.Fatalf("no zhipu row in %v", got)
	}
}

func TestRoster_omits_a_model_too_narrow_for_claude_code(t *testing.T) {
	for _, id := range models(Roster(rosterEnv(t))) {
		if strings.HasSuffix(id, "/glm-4.5-air") {
			t.Fatalf("131072-token model was offered: %s", id)
		}
	}
}

func TestRoster_labels_a_row_with_its_source(t *testing.T) {
	for _, row := range Roster(rosterEnv(t)) {
		if row.Model == "wisp/acct.personal/claude-opus-5" && !strings.Contains(row.Label, "Personal") {
			t.Fatalf("label %q does not name the account", row.Label)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/allin/ -run TestRoster -v`
Expected: FAIL — `undefined: Roster`.

- [ ] **Step 3: Write minimal implementation**

```go
package allin

import (
	"fmt"
	"os"
	"strings"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// minRosterContext is the narrowest window worth offering. Claude Code's own
// floor is ~20k tokens before a conversation starts, and a profile reserves a
// quarter of the window for the reply, so anything tighter cannot finish a task.
const minRosterContext = 200000

// Env names the four files the roster is built from. They are the same files
// the account switcher and the subscription modal already own.
type Env struct {
	AccountsList string
	AccountsDir  string
	ConfigsList  string
	ConfigsDir   string
}

// Row is one entry of the settings key `modelPicker.options`.
type Row struct {
	Model       string `json:"model"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	BehavesAs   string `json:"behavesAs,omitempty"`
}

// claudeModel is one first-party model offered for every Claude login.
type claudeModel struct {
	id      string
	label   string
	wide    bool // also offer a [1m] row
}

// claudeLineup is pinned rather than discovered: Claude Code ships no catalog a
// third party can read, and an id it does not know still routes fine.
var claudeLineup = []claudeModel{
	{"claude-opus-5", "Opus 5", true},
	{"claude-sonnet-5", "Sonnet 5", true},
	{"claude-fable-5-1", "Fable 5.1", false},
	{"claude-haiku-4-5-20251001", "Haiku 4.5", false},
}

// Roster builds every picker row, accounts first, then configured providers.
// A source that cannot be read contributes nothing rather than failing the set.
func Roster(env Env) []Row {
	rows := accountRows(env)
	return append(rows, configRows(env)...)
}

func accountRows(env Env) []Row {
	accounts := []struct{ label, dir string }{{"Default", "default"}}
	for _, line := range readLines(env.AccountsList) {
		label, dir, ok := strings.Cut(line, ":")
		if !ok || label == "" || dir == "" {
			continue
		}
		accounts = append(accounts, struct{ label, dir string }{label, dir})
	}
	var rows []Row
	for _, account := range accounts {
		for _, model := range claudeLineup {
			rows = append(rows, Row{
				Model:       fmt.Sprintf("wisp/acct.%s/%s", account.dir, model.id),
				Label:       account.label + " · " + model.label,
				Description: "Claude subscription: " + account.label,
			})
			if model.wide {
				rows = append(rows, Row{
					Model:       fmt.Sprintf("wisp/acct.%s/%s[1m]", account.dir, model.id),
					Label:       account.label + " · " + model.label + " (1M)",
					Description: "Claude subscription: " + account.label,
				})
			}
		}
	}
	return rows
}

func configRows(env Env) []Row {
	var rows []Row
	for _, config := range claudeconfig.Load(env.ConfigsList) {
		if !claudeconfig.ConfigReady(env.ConfigsDir, config) {
			continue
		}
		provider := claudeconfig.ProviderForConfig(env.ConfigsDir, config)
		source := strings.TrimSuffix(config.File, ".json")
		for _, model := range providerModels(env, config, provider) {
			id := model.ID
			suffix := ""
			if model.Context >= 1000000 {
				suffix = "[1m]"
			} else if model.Context != 0 && model.Context < minRosterContext {
				continue
			}
			rows = append(rows, Row{
				Model:       fmt.Sprintf("wisp/cfg.%s/%s%s", source, id, suffix),
				Label:       config.Name + " · " + id,
				Description: provider.Name,
			})
		}
	}
	return rows
}

// providerModels returns the catalog's models, or the single model the user
// supplied for a provider that ships none.
func providerModels(env Env, config claudeconfig.Config, provider claudeconfig.Provider) []claudeconfig.Model {
	if provider.SuppliesOwnModel() {
		id := claudeconfig.ReadCustomModel(env.ConfigsDir, config.File)
		if id == "" {
			return nil
		}
		// Window unknown here; the profile already declares it, and a row with
		// no declared context is offered at Claude Code's flat 200000.
		return []claudeconfig.Model{{ID: id}}
	}
	return provider.Models
}

func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/allin/ -run TestRoster -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/allin/roster.go internal/allin/roster_test.go
git commit -m "feat(allin): build picker rows from every login and ready provider"
```

---

### Task 3: Резолвер кредов и чтение Keychain

**Files:**
- Create: `internal/allin/credential.go`
- Create: `internal/allin/keychain_darwin.go`
- Create: `internal/allin/keychain_other.go`
- Modify: `test/bash/idle_sound_ownership_test.go` (запись в `auditedProductionProcessCalls`)
- Test: `internal/allin/credential_test.go`

**Interfaces:**
- Consumes: `Target` (Task 1), `Env` (Task 2), `claudeconfig.ReadAPIKey`, `claudeconfig.ReadBaseURL`.
- Produces: `type Credential struct { BaseURL, Header, Value string }`; `type Resolver interface { Resolve(Target) (Credential, error) }`; `type FileResolver struct { Env Env; Token func(configDir string) (string, error) }`; `func NewResolver(env Env) *FileResolver`; `func KeychainService(configDir string) string`; `var ErrStaleAccount = errors.New(...)`.

- [ ] **Step 1: Write the failing test**

```go
package allin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestKeychainService_names_the_default_login_without_a_suffix(t *testing.T) {
	if got := KeychainService(""); got != "Claude Code-credentials" {
		t.Fatalf("got %q", got)
	}
}

func TestKeychainService_derives_the_suffix_from_the_config_dir(t *testing.T) {
	// sha256("/Users/jackuait/.config/wisp-deck/claude-accounts/personal")[:8]
	got := KeychainService("/Users/jackuait/.config/wisp-deck/claude-accounts/personal")
	if got != "Claude Code-credentials-7646b36d" {
		t.Fatalf("got %q", got)
	}
}

func TestResolve_hands_an_account_row_its_own_oauth_token(t *testing.T) {
	env := rosterEnv(t)
	resolver := NewResolver(env)
	resolver.Token = func(configDir string) (string, error) { return "oat-personal", nil }
	got, err := resolver.Resolve(Target{Kind: KindAccount, Source: "personal", Model: "claude-opus-5"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Header != "Authorization" || got.Value != "Bearer oat-personal" {
		t.Fatalf("got %+v", got)
	}
	if got.BaseURL != anthropicUpstream {
		t.Fatalf("base %q", got.BaseURL)
	}
}

func TestResolve_hands_a_config_row_its_profile_key_and_endpoint(t *testing.T) {
	env := rosterEnv(t)
	got, err := NewResolver(env).Resolve(Target{Kind: KindConfig, Source: "zhipu-glm", Model: "glm-4.7"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != "Bearer k" || got.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolve_reports_a_stale_account_by_name(t *testing.T) {
	env := rosterEnv(t)
	resolver := NewResolver(env)
	resolver.Token = func(string) (string, error) { return "", errors.New("not found") }
	_, err := resolver.Resolve(Target{Kind: KindAccount, Source: "personal"})
	if !errors.Is(err, ErrStaleAccount) {
		t.Fatalf("err = %v", err)
	}
}

func TestResolve_refuses_a_source_that_escapes_its_directory(t *testing.T) {
	env := rosterEnv(t)
	if _, err := NewResolver(env).Resolve(Target{Kind: KindConfig, Source: "../../etc/passwd"}); err == nil {
		t.Fatal("traversal accepted")
	}
	_ = os.WriteFile(filepath.Join(env.ConfigsDir, "x.json"), []byte("{}"), 0o600)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/allin/ -run 'TestKeychainService|TestResolve' -v`
Expected: FAIL — `undefined: KeychainService`.

- [ ] **Step 3: Write minimal implementation**

`internal/allin/credential.go`:

```go
package allin

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// anthropicUpstream is where a Claude login's turn goes. A login has no
// endpoint of its own — only a credential.
const anthropicUpstream = "https://api.anthropic.com"

// ErrStaleAccount marks a login whose token could not be read. It is
// deterministic, so the router must surface it as 400: Claude Code retries a
// 401 about eleven times before giving up.
var ErrStaleAccount = errors.New("allin: account credential unavailable")

// Credential is the endpoint and auth header one target needs.
type Credential struct {
	BaseURL string
	Header  string
	Value   string
}

// Resolver answers what a parsed picker row should be sent with.
type Resolver interface {
	Resolve(Target) (Credential, error)
}

// FileResolver reads the same files the account switcher owns. Token is a seam
// so tests never touch the real Keychain.
type FileResolver struct {
	Env   Env
	Token func(configDir string) (string, error)
}

func NewResolver(env Env) *FileResolver {
	return &FileResolver{Env: env, Token: keychainToken}
}

func (r *FileResolver) Resolve(target Target) (Credential, error) {
	if err := validSource(target.Source); err != nil {
		return Credential{}, err
	}
	switch target.Kind {
	case KindAccount:
		configDir := ""
		if target.Source != "default" {
			configDir = filepath.Join(r.Env.AccountsDir, target.Source)
		}
		token, err := r.Token(configDir)
		if err != nil || token == "" {
			return Credential{}, fmt.Errorf("%w: %s", ErrStaleAccount, target.Source)
		}
		return Credential{BaseURL: anthropicUpstream, Header: "Authorization", Value: "Bearer " + token}, nil
	case KindConfig:
		file := target.Source + ".json"
		key := claudeconfig.ReadAPIKey(r.Env.ConfigsDir, file)
		base := claudeconfig.ReadBaseURL(r.Env.ConfigsDir, file)
		if key == "" || base == "" {
			return Credential{}, fmt.Errorf("allin: profile %q is not ready", target.Source)
		}
		return Credential{BaseURL: base, Header: "Authorization", Value: "Bearer " + key}, nil
	}
	return Credential{}, errors.New("allin: session target needs no credential")
}

// validSource keeps a model id from naming a path. The id comes off the wire,
// so a "../" source would otherwise read any file the user can read.
func validSource(source string) error {
	if source == "" {
		return errors.New("allin: empty source")
	}
	if strings.ContainsAny(source, "/\\") || strings.Contains(source, "..") {
		return fmt.Errorf("allin: source %q is not a single name", source)
	}
	return nil
}
```

`internal/allin/keychain_darwin.go`:

```go
//go:build darwin

package allin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// KeychainService names the Keychain entry Claude Code writes for one config
// dir. The suffix is sha256 of the directory path, first eight hex characters;
// the default login (no CLAUDE_CONFIG_DIR) has no suffix.
func KeychainService(configDir string) string {
	const base = "Claude Code-credentials"
	if configDir == "" {
		return base
	}
	sum := sha256.Sum256([]byte(configDir))
	return base + "-" + hex.EncodeToString(sum[:])[:8]
}

func keychainToken(configDir string) (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", KeychainService(configDir), "-w").Output()
	if err != nil {
		return "", err
	}
	var blob struct {
		OAuth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &blob); err != nil {
		return "", fmt.Errorf("allin: parse keychain blob: %w", err)
	}
	return blob.OAuth.AccessToken, nil
}
```

`internal/allin/keychain_other.go`:

```go
//go:build !darwin

package allin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

func KeychainService(configDir string) string {
	const base = "Claude Code-credentials"
	if configDir == "" {
		return base
	}
	sum := sha256.Sum256([]byte(configDir))
	return base + "-" + hex.EncodeToString(sum[:])[:8]
}

func keychainToken(string) (string, error) {
	return "", errors.New("allin: Keychain is only readable on darwin")
}
```

- [ ] **Step 4: Register the spawn, then run the tests**

Добавить в map, который возвращает `auditedProductionProcessCalls()` в `test/bash/idle_sound_ownership_test.go`, сохраняя выравнивание соседних строк:

```go
`internal/allin/keychain_darwin.go:keychainToken:exec.Command("security", "find-generic-password", "-s", KeychainService(configDir), "-w")`: 1,
```

Run: `go test ./internal/allin/ -run 'TestKeychainService|TestResolve' -v`
Expected: PASS.

Run: `go test ./test/bash/ -run TestProductionProcessCalls -v`
Expected: PASS — незарегистрированный спавн не найден.

- [ ] **Step 5: Commit**

```bash
git add internal/allin/credential.go internal/allin/keychain_darwin.go \
        internal/allin/keychain_other.go internal/allin/credential_test.go \
        test/bash/idle_sound_ownership_test.go
git commit -m "feat(allin): resolve each row's endpoint and credential"
```

---

### Task 4: HTTP-роутер

**Files:**
- Create: `internal/allin/proxy.go`
- Test: `internal/allin/proxy_test.go`

**Interfaces:**
- Consumes: `Route` (Task 1), `Resolver`/`Credential`/`ErrStaleAccount` (Task 3).
- Produces: `func NewHandler(resolver Resolver, sessionUpstream string) http.Handler`.

- [ ] **Step 1: Write the failing test**

```go
package allin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeResolver struct {
	credential Credential
	err        error
}

func (f fakeResolver) Resolve(Target) (Credential, error) { return f.credential, f.err }

func upstreamRecorder(t *testing.T, status int) (*httptest.Server, *http.Request, *[]byte) {
	t.Helper()
	var seen *http.Request
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		seen = r.Clone(r.Context())
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)
	return server, seen, &body
}

func TestHandler_swaps_the_credential_and_strips_the_row_prefix(t *testing.T) {
	var gotAuth, gotBeta string
	var gotBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBeta = r.Header.Get("Anthropic-Beta")
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{credential: Credential{
		BaseURL: upstream.URL, Header: "Authorization", Value: "Bearer swapped",
	}}, "http://unused.invalid")

	request := httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5[1m]"}`))
	request.Header.Set("Authorization", "Bearer session")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotAuth != "Bearer swapped" {
		t.Fatalf("auth %q", gotAuth)
	}
	if !strings.Contains(gotBeta, "context-1m-2025-08-07") {
		t.Fatalf("beta %q", gotBeta)
	}
	var sent map[string]any
	_ = json.Unmarshal(gotBody, &sent)
	if sent["model"] != "claude-opus-5" {
		t.Fatalf("model %v", sent["model"])
	}
}

func TestHandler_leaves_a_session_row_untouched(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{err: errors.New("must not be called")}, upstream.URL)
	request := httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"claude-opus-5"}`))
	request.Header.Set("Authorization", "Bearer session")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotAuth != "Bearer session" {
		t.Fatalf("auth %q", gotAuth)
	}
}

func TestHandler_reports_a_stale_account_as_400_not_401(t *testing.T) {
	handler := NewHandler(fakeResolver{err: ErrStaleAccount}, "http://unused.invalid")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/messages",
		strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5"}`)))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "personal") {
		t.Fatalf("body does not name the account: %s", recorder.Body.String())
	}
}

func TestHandler_routes_a_count_tokens_request_the_same_way(t *testing.T) {
	var gotPath, gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	handler := NewHandler(fakeResolver{credential: Credential{
		BaseURL: upstream.URL, Header: "Authorization", Value: "Bearer swapped",
	}}, "http://unused.invalid")
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost,
		"/v1/messages/count_tokens", strings.NewReader(`{"model":"wisp/acct.personal/claude-opus-5"}`)))

	if gotPath != "/v1/messages/count_tokens" || gotAuth != "Bearer swapped" {
		t.Fatalf("path %q auth %q", gotPath, gotAuth)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/allin/ -run TestHandler -v`
Expected: FAIL — `undefined: NewHandler`.

- [ ] **Step 3: Write minimal implementation**

```go
package allin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// maxRouteBytes caps the body read to re-address a request. A larger one is
// forwarded unread on the session's own credential rather than truncated.
const maxRouteBytes = 8 << 20

// NewHandler routes each request by the model its body names. A row this build
// cannot place goes to sessionUpstream on the session's own credential, so an
// unrecognised id costs a turn nothing.
func NewHandler(resolver Resolver, sessionUpstream string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base, body := sessionUpstream, []byte(nil)
		if r.Method == http.MethodPost && r.Body != nil && r.ContentLength <= maxRouteBytes {
			raw, err := io.ReadAll(io.LimitReader(r.Body, maxRouteBytes))
			_ = r.Body.Close()
			if err == nil {
				body = raw
			}
		}

		var payload map[string]any
		if body != nil {
			_ = json.Unmarshal(body, &payload)
		}
		model, _ := payload["model"].(string)
		target := Route(model)

		if target.Kind != KindSession {
			credential, err := resolver.Resolve(target)
			if err != nil {
				writeRoutingError(w, target, err)
				return
			}
			base = credential.BaseURL
			payload["model"] = target.Model
			if rewritten, err := json.Marshal(payload); err == nil {
				body = rewritten
			}
			r.Header.Set(credential.Header, credential.Value)
			if credential.Header == "Authorization" {
				r.Header.Del("X-Api-Key")
			}
			if target.Want1M {
				addBeta(r.Header, "context-1m-2025-08-07")
			}
		}

		if body != nil {
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
			r.Header.Set("Content-Length", fmt.Sprint(len(body)))
		}
		newReverseProxy(base).ServeHTTP(w, r)
	})
}

func newReverseProxy(upstream string) *httputil.ReverseProxy {
	target, err := url.Parse(upstream)
	if err != nil {
		return &httputil.ReverseProxy{Director: func(*http.Request) {}}
	}
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme, req.URL.Host = target.Scheme, target.Host
			req.URL.Path = strings.TrimSuffix(target.Path, "/") + req.URL.Path
			// Gateways route on Host; the loopback name reaches no virtual host.
			req.Host = target.Host
			// A compressed body is bytes the next hop cannot read.
			req.Header.Set("Accept-Encoding", "identity")
		},
		// Each write forwarded as it arrives: buffering swallows the keep-alive
		// bytes that keep Claude Code's stall watchdog from replaying a turn.
		FlushInterval: -1,
	}
}

func addBeta(header http.Header, value string) {
	current := header.Get("Anthropic-Beta")
	if strings.Contains(current, value) {
		return
	}
	if current == "" {
		header.Set("Anthropic-Beta", value)
		return
	}
	header.Set("Anthropic-Beta", current+","+value)
}

// writeRoutingError answers in Anthropic's own envelope, always with 400.
// Claude Code retries 401 and 5xx about eleven times, and every one of these is
// deterministic — the same request would fail again the same way.
func writeRoutingError(w http.ResponseWriter, target Target, err error) {
	message := fmt.Sprintf("wisp-deck: %v", err)
	if errors.Is(err, ErrStaleAccount) {
		message = fmt.Sprintf(
			"wisp-deck: the login %q has no usable credential. Open it once "+
				"(a wisp-deck tab on that account) so Claude refreshes its token, then retry.",
			target.Source)
	}
	body, _ := json.Marshal(map[string]any{
		"type":  "error",
		"error": map[string]string{"type": "invalid_request_error", "message": message},
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write(body)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/allin/ -v`
Expected: PASS — все тесты Task 1-4.

- [ ] **Step 5: Commit**

```bash
git add internal/allin/proxy.go internal/allin/proxy_test.go
git commit -m "feat(allin): route each turn to its own endpoint and credential"
```

---

### Task 5: Профиль All-In и запись пикера

**Files:**
- Create: `internal/allin/profile.go`
- Test: `internal/allin/profile_test.go`

**Interfaces:**
- Consumes: `Roster`/`Env`/`Row` (Task 2), `claudeconfig.Load`, `claudeconfig.Add`.
- Produces: `const ProfileName = "All-In"`; `func EnsureProfile(env Env, listFile, configsDir string) (string, error)` — возвращает имя файла профиля. Вызывается из Task 9 (`ensure-allin`); без неё профиль не создаётся никогда.

- [ ] **Step 1: Write the failing test**

```go
package allin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readPicker(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	picker, _ := settings["modelPicker"].(map[string]any)
	if picker == nil {
		t.Fatalf("no modelPicker in %s", data)
	}
	return picker
}

func TestEnsureProfile_creates_the_profile_and_registers_it(t *testing.T) {
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	picker := readPicker(t, filepath.Join(env.ConfigsDir, file))
	options, _ := picker["options"].([]any)
	if len(options) == 0 {
		t.Fatal("no rows written")
	}
	if picker["replaceBuiltInOptions"] != true {
		t.Fatalf("built-ins not replaced: %v", picker)
	}
	registered := false
	for _, config := range readLines(listFile) {
		if config == ProfileName+":"+file {
			registered = true
		}
	}
	if !registered {
		t.Fatalf("not registered in %s", listFile)
	}
}

func TestEnsureProfile_refreshes_rows_without_a_second_registration(t *testing.T) {
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	first, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("%q != %q", first, second)
	}
	if lines := readLines(listFile); len(lines) != 1 {
		t.Fatalf("registered %d times: %v", len(lines), lines)
	}
}

func TestEnsureProfile_keeps_every_other_key_in_the_profile(t *testing.T) {
	env := rosterEnv(t)
	listFile := filepath.Join(t.TempDir(), "claude-configs.list")
	file, err := EnsureProfile(env, listFile, env.ConfigsDir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(env.ConfigsDir, file)
	data, _ := os.ReadFile(path)
	var settings map[string]any
	_ = json.Unmarshal(data, &settings)
	settings["statusLine"] = "keep me"
	patched, _ := json.Marshal(settings)
	_ = os.WriteFile(path, patched, 0o600)

	if _, err := EnsureProfile(env, listFile, env.ConfigsDir); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	_ = json.Unmarshal(data, &settings)
	if settings["statusLine"] != "keep me" {
		t.Fatalf("unrelated key lost: %s", data)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/allin/ -run TestEnsureProfile -v`
Expected: FAIL — `undefined: EnsureProfile`.

- [ ] **Step 3: Write minimal implementation**

```go
package allin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// ProfileName is the display name of the generated profile. It is matched
// verbatim in the configs list, so renaming it orphans the existing one.
const ProfileName = "All-In"

// EnsureProfile creates the All-In profile when absent and rewrites its picker
// rows every call — logins and providers come and go, and a stale roster offers
// models the machine can no longer reach.
//
// Only modelPicker is written. Every other key in the file is the user's, and
// the launch overlay copies the whole object.
func EnsureProfile(env Env, listFile, configsDir string) (string, error) {
	file := existingProfile(listFile)
	if file == "" {
		created, err := claudeconfig.Add(listFile, configsDir, ProfileName)
		if err != nil {
			return "", err
		}
		file = created
	}

	path := filepath.Join(configsDir, file)
	settings := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	rows := Roster(env)
	options := make([]Row, 0, len(rows))
	options = append(options, rows...)
	settings["modelPicker"] = map[string]any{
		// Every row names its account explicitly, so the built-in lineup would
		// only add rows whose credential is ambiguous.
		"replaceBuiltInOptions": true,
		"options":               options,
	}

	encoded, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	// Published by rename: a reader must never see half a settings file.
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(encoded, '\n'), 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(temporary, path); err != nil {
		return "", err
	}
	return file, nil
}

func existingProfile(listFile string) string {
	for _, config := range claudeconfig.Load(listFile) {
		if strings.EqualFold(config.Name, ProfileName) {
			return config.File
		}
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/allin/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/allin/profile.go internal/allin/profile_test.go
git commit -m "feat(allin): generate and refresh the All-In profile's picker"
```

---

### Task 6: CLI-обёртка `claude-allin`

**Files:**
- Create: `cmd/wisp-deck-tui/claude_allin.go`
- Modify: `test/bash/idle_sound_ownership_test.go` (второй `exec.Command`)
- Test: `cmd/wisp-deck-tui/claude_allin_test.go`

**Interfaces:**
- Consumes: `allin.NewHandler`, `allin.NewResolver`, `allin.Env` (Tasks 2-4); `rolefix.PointSettingsAt` для переписывания overlay'я.
- Produces: cobra-команда `claude-allin --settings PATH --accounts-list … --accounts-dir … --configs-list … --configs-dir … -- COMMAND [ARG...]`.

- [ ] **Step 1: Write the failing test**

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeAllIn_points_the_overlay_at_the_local_router(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(settings,
		[]byte(`{"env":{"ANTHROPIC_BASE_URL":"https://api.anthropic.com"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var seen string
	command := newClaudeAllInCommand(func([]string) error {
		data, _ := os.ReadFile(settings)
		var parsed struct {
			Env map[string]string `json:"env"`
		}
		_ = json.Unmarshal(data, &parsed)
		seen = parsed.Env["ANTHROPIC_BASE_URL"]
		return nil
	})
	command.SetArgs([]string{"--settings", settings, "--", "true"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !hasLoopbackPrefix(seen) {
		t.Fatalf("child saw %q, not a loopback router", seen)
	}
}

func TestClaudeAllIn_runs_the_child_when_the_overlay_cannot_be_read(t *testing.T) {
	ran := false
	command := newClaudeAllInCommand(func([]string) error { ran = true; return nil })
	command.SetArgs([]string{"--settings", filepath.Join(t.TempDir(), "absent.json"), "--", "true"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("child never ran")
	}
}

func hasLoopbackPrefix(url string) bool {
	return len(url) > 17 && url[:17] == "http://127.0.0.1:"
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/wisp-deck-tui/ -run TestClaudeAllIn -v`
Expected: FAIL — `undefined: newClaudeAllInCommand`.

- [ ] **Step 3: Write minimal implementation**

```go
package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/jackuait/wisp-deck/internal/allin"
	"github.com/jackuait/wisp-deck/internal/rolefix"
)

func init() {
	rootCmd.AddCommand(newClaudeAllInCommandWithExit(runClaudeAllInChild, os.Exit))
}

func newClaudeAllInCommand(run claudeRolefixRunner) *cobra.Command {
	return newClaudeAllInCommandWithExit(run, func(int) {})
}

// newClaudeAllInCommandWithExit wraps one Claude launch in a loopback router
// that sends each turn to the subscription its picker row names.
//
// Nothing here may cost the user their session: an unreadable overlay, or one
// declaring no endpoint, runs the child exactly as it was going to run anyway.
func newClaudeAllInCommandWithExit(run claudeRolefixRunner, exit func(int)) *cobra.Command {
	var settingsPath string
	var env allin.Env
	command := &cobra.Command{
		Use:          "claude-allin --settings PATH -- COMMAND [ARG...]",
		Short:        "Route one Claude launch across every configured subscription",
		Hidden:       true,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, argv []string) error {
			if run == nil {
				return errors.New("child runner is unavailable")
			}
			finish := func(err error) error {
				var code exitCodeError
				if errors.As(err, &code) {
					if exit != nil {
						exit(int(code))
					}
					return nil
				}
				return err
			}
			upstream, err := rolefix.UpstreamFromSettings(settingsPath)
			if err != nil {
				return finish(run(argv))
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return finish(run(argv))
			}
			defer func() { _ = listener.Close() }()

			server := &http.Server{Handler: allin.NewHandler(allin.NewResolver(env), upstream)}
			go func() { _ = server.Serve(listener) }()
			defer func() { _ = server.Close() }()

			if err := rolefix.PointSettingsAt(settingsPath, fmt.Sprintf("http://%s", listener.Addr().String())); err != nil {
				return finish(run(argv))
			}
			return finish(run(argv))
		},
	}
	flags := command.Flags()
	flags.StringVar(&settingsPath, "settings", "", "launch settings overlay to point at the router")
	flags.StringVar(&env.AccountsList, "accounts-list", "", "name:dir list of Claude logins")
	flags.StringVar(&env.AccountsDir, "accounts-dir", "", "directory holding each login's config dir")
	flags.StringVar(&env.ConfigsList, "configs-list", "", "name:file list of subscription profiles")
	flags.StringVar(&env.ConfigsDir, "configs-dir", "", "directory holding the profile settings files")
	return command
}

func runClaudeAllInChild(argv []string) error {
	child := exec.Command(argv[0], argv[1:]...)
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := child.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exitCodeError(exit.ExitCode())
		}
		return err
	}
	return nil
}
```

- [ ] **Step 4: Register the spawn, then run the tests**

Добавить в `auditedProductionProcessCalls()`:

```go
`cmd/wisp-deck-tui/claude_allin.go:runClaudeAllInChild:exec.Command(argv[0], argv[1:]...)`: 1,
```

Run: `go test ./cmd/wisp-deck-tui/ -run TestClaudeAllIn -v`
Expected: PASS.

Run: `go test ./test/bash/ -run TestProductionProcessCalls -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/wisp-deck-tui/claude_allin.go cmd/wisp-deck-tui/claude_allin_test.go \
        test/bash/idle_sound_ownership_test.go
git commit -m "feat(allin): wrap a launch in the cross-subscription router"
```

---

### Task 7: Привязка к запуску

**Files:**
- Modify: `lib/tmux-session.sh:148` (там же, где выбирается `claude-rolefix`)
- Test: `test/bash/allin_launch_test.go`

**Interfaces:**
- Consumes: команду `claude-allin` (Task 6), `ProfileName` (Task 5).
- Produces: функцию `gt_claude_launch_wrapper <settings_path> <provider_marker> <config_name>`, печатающую префикс argv.

- [ ] **Step 1: Write the failing test**

```go
package bash_test

import "testing"

func TestClaudeLaunchWrapper_routes_the_all_in_profile_through_the_router(t *testing.T) {
	out, code := runBashFunc(t, "lib/tmux-session.sh", "gt_claude_launch_wrapper",
		[]string{"/tmp/overlay.json", "", "All-In"}, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "claude-allin")
	assertNotContains(t, out, "claude-rolefix")
}

func TestClaudeLaunchWrapper_keeps_featherless_on_the_role_repair_proxy(t *testing.T) {
	out, code := runBashFunc(t, "lib/tmux-session.sh", "gt_claude_launch_wrapper",
		[]string{"/tmp/overlay.json", "featherless", "Featherless Qwen"}, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "claude-rolefix")
	assertNotContains(t, out, "claude-allin")
}

func TestClaudeLaunchWrapper_wraps_nothing_for_an_ordinary_profile(t *testing.T) {
	out, code := runBashFunc(t, "lib/tmux-session.sh", "gt_claude_launch_wrapper",
		[]string{"/tmp/overlay.json", "", "Zhipu GLM"}, nil)
	assertExitCode(t, code, 0)
	assertNotContains(t, out, "claude-allin")
	assertNotContains(t, out, "claude-rolefix")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./test/bash/ -run TestClaudeLaunchWrapper -v -timeout 20m`
Expected: FAIL — функция не найдена.

- [ ] **Step 3: Write minimal implementation**

В `lib/tmux-session.sh`, рядом с существующей сборкой `raw` на строке 148:

```bash
# gt_claude_launch_wrapper <settings_path> <provider_marker> <config_name>
#
# Print the argv prefix that wraps one Claude launch, or nothing. The All-In
# profile routes every turn by its picker row, so it takes the cross-account
# router; Featherless keeps the role-repair proxy. Both are mutually exclusive:
# the router already forwards to whatever endpoint a row names.
gt_claude_launch_wrapper() {
  local settings_path="$1" provider_marker="$2" config_name="$3"
  local config_root="${WISP_DECK_CONFIG_DIR:-$HOME/.config/wisp-deck}"

  if [ "$config_name" = "All-In" ]; then
    printf 'wisp-deck-tui claude-allin --settings %s' "$settings_path"
    printf ' --accounts-list %s/claude-accounts.list' "$config_root"
    printf ' --accounts-dir %s/claude-accounts' "$config_root"
    printf ' --configs-list %s/claude-configs.list' "$config_root"
    printf ' --configs-dir %s/claude-configs --' "$config_root"
    return 0
  fi
  if [ "$provider_marker" = "featherless" ]; then
    printf 'wisp-deck-tui claude-rolefix --settings %s --' "$settings_path"
    return 0
  fi
  return 0
}
```

Затем в месте сборки `raw` (строка 148) заменить безусловный `claude-rolefix` на вызов этой функции, сохранив существующее кавычивание через `${settings_q}`.

- [ ] **Step 4: Run tests and shellcheck**

Run: `go test ./test/bash/ -run TestClaudeLaunchWrapper -v -timeout 20m`
Expected: PASS.

Run: `shellcheck lib/tmux-session.sh`
Expected: без предупреждений.

- [ ] **Step 5: Commit**

```bash
git add lib/tmux-session.sh test/bash/allin_launch_test.go
git commit -m "feat(allin): launch the All-In profile through the router"
```

---

### Task 8: Документация

**Files:**
- Modify: `CLAUDE.md` (новый раздел после «A Featherless pane runs behind a request repair proxy»)

- [ ] **Step 1: Написать раздел**

Заголовок: `### All-In routes one session across every subscription, by the model id`.
Раздел обязан зафиксировать ровно то, что измерено, и почему каждое решение такое:

- грамматика `wisp/<source>/<model>`, и почему режем только по первым двум сегментам (Featherless-id содержит слэш);
- `[1m]` — единственный способ дать строке 1M-окно; `behavesAs` окно НЕ переносит (замерено);
- протухший токен отдаётся **400**, а не 401, потому что Claude Code повторяет 401 около 11 раз;
- `replaceBuiltInOptions: true` — иначе встроенные строки идут на неявный логин сессии;
- модели уже 200000 не предлагаются;
- роутер и rolefix взаимоисключающи;
- рефреша токенов нет: аккаунт лечится ручным открытием;
- прогревочный запрос идёт на модель по умолчанию, поэтому маршрут по умолчанию обязателен;
- ссылка на `docs/all-in-model-router.md` и на охраняющие тесты.

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: record how All-In routes a session across subscriptions"
```

---

### Task 9: Sweep, который создаёт и обновляет профиль

**Files:**
- Modify: `cmd/wisp-deck-tui/claude_config.go` (новая подкоманда рядом с `ensure-budget:81`)
- Modify: `bin/wisp-deck:225` (вызов рядом с существующими sweep'ами)
- Test: `cmd/wisp-deck-tui/claude_config_allin_test.go`

**Interfaces:**
- Consumes: `allin.EnsureProfile`, `allin.Env`, `allin.ProfileName` (Task 5).
- Produces: подкоманду `wisp-deck-tui claude-config ensure-allin --dir DIR --list FILE --accounts-list FILE --accounts-dir DIR`.

Профиль обязан пересобираться на каждой установке: логины и провайдеры добавляются
и удаляются, а устаревший ростер предлагает модели, до которых машина уже не достаёт.
Это тот же приём, которым `ensure-budget` чинит профили, созданные до появления ключа.

- [ ] **Step 1: Write the failing test**

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureAllIn_creates_the_profile_on_a_fresh_machine(t *testing.T) {
	root := t.TempDir()
	configs := filepath.Join(root, "claude-configs")
	accounts := filepath.Join(root, "claude-accounts")
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(accounts, 0o755); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(root, "claude-configs.list")
	accountsList := filepath.Join(root, "claude-accounts.list")
	if err := os.WriteFile(accountsList, []byte("Personal:personal\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	command := newClaudeConfigCommand()
	command.SetArgs([]string{"ensure-allin", "--dir", configs, "--list", list,
		"--accounts-list", accountsList, "--accounts-dir", accounts})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(configs)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no profile written: %v %v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(configs, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if settings["modelPicker"] == nil {
		t.Fatalf("profile has no picker: %s", data)
	}
}

func TestEnsureAllIn_is_idempotent(t *testing.T) {
	root := t.TempDir()
	configs := filepath.Join(root, "claude-configs")
	accounts := filepath.Join(root, "claude-accounts")
	_ = os.MkdirAll(configs, 0o755)
	_ = os.MkdirAll(accounts, 0o755)
	list := filepath.Join(root, "claude-configs.list")
	accountsList := filepath.Join(root, "claude-accounts.list")
	_ = os.WriteFile(accountsList, []byte(""), 0o600)

	for i := 0; i < 2; i++ {
		command := newClaudeConfigCommand()
		command.SetArgs([]string{"ensure-allin", "--dir", configs, "--list", list,
			"--accounts-list", accountsList, "--accounts-dir", accounts})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(configs)
	if len(entries) != 1 {
		t.Fatalf("wrote %d profiles, want 1", len(entries))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/wisp-deck-tui/ -run TestEnsureAllIn -v`
Expected: FAIL — `unknown command "ensure-allin"`.

- [ ] **Step 3: Write minimal implementation**

Зарегистрировать подкоманду там же, где объявлена `ensure-budget` (`cmd/wisp-deck-tui/claude_config.go:81`):

```go
func newEnsureAllInCommand() *cobra.Command {
	var env allin.Env
	var listFile string
	command := &cobra.Command{
		Use:   "ensure-allin",
		Short: "Create or refresh the All-In profile's model picker",
		RunE: func(_ *cobra.Command, _ []string) error {
			env.ConfigsList = listFile
			// A machine with no source to route between has nothing to offer,
			// and an All-In row would only duplicate the built-in lineup.
			if rows := allin.Roster(env); len(rows) == 0 {
				return nil
			}
			_, err := allin.EnsureProfile(env, listFile, env.ConfigsDir)
			return err
		},
	}
	flags := command.Flags()
	flags.StringVar(&env.ConfigsDir, "dir", "", "directory holding the profile settings files")
	flags.StringVar(&listFile, "list", "", "name:file list of subscription profiles")
	flags.StringVar(&env.AccountsList, "accounts-list", "", "name:dir list of Claude logins")
	flags.StringVar(&env.AccountsDir, "accounts-dir", "", "directory holding each login's config dir")
	return command
}
```

Затем добавить `command.AddCommand(newEnsureAllInCommand())` рядом с уже
зарегистрированной `ensure-budget`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/wisp-deck-tui/ -run TestEnsureAllIn -v`
Expected: PASS.

- [ ] **Step 5: Wire the sweep into the installer**

В `bin/wisp-deck`, сразу после строки 229 (`ensure-watchdog`):

```bash
  wisp-deck-tui claude-config ensure-allin --dir "$CONFIGS_DIR/claude-configs"     --list "$CONFIGS_DIR/claude-configs.list"     --accounts-list "$CONFIGS_DIR/claude-accounts.list"     --accounts-dir "$CONFIGS_DIR/claude-accounts" >/dev/null 2>&1 || true
```

Run: `shellcheck bin/wisp-deck`
Expected: без предупреждений.

- [ ] **Step 6: Commit**

```bash
git add cmd/wisp-deck-tui/claude_config.go cmd/wisp-deck-tui/claude_config_allin_test.go bin/wisp-deck
git commit -m "feat(allin): create and refresh the All-In profile on every install"
```

---

## Отложено сознательно (не в этом плане)

- **GPT-строки через `gptbridge`.** Мост не HTTP-эндпоинт с ключом: его поднимает `gptbridge.StartLoopbackServer(executor, key, ServerOptions{})`, и `Engine.Execute` валидирует `model` по allowlist (`internal/gptbridge/engine.go:140`), поэтому id надо переписывать в голый codex-id. Это отдельная задача поверх Task 4: ленивый старт моста и `Credential` с его loopback-URL.
- **Автофейловер на 429** — решено не делать в v1.
- **Refresh протухших токенов** — решено не делать в v1.
