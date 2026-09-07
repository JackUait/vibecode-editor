# All-In: единый пикер моделей всех подписок — спека

Статус: измерено на живых панелях 2026-09-07, Claude Code 2.1.263.
Полный лог замеров и прототип роутера: см. раздел «Измерения» ниже.

## Задача

Один аккаунт `All-In`, в чьём `/model` видны модели **всех** подключённых источников:
каждого Claude-логина (Default, Personal, …) и каждого subscription-профиля
(Zhipu, MiMo, ChatGPT, Kimi, Qwen, Featherless). Переключение между ними —
**без перезапуска панели и без потери диалога**.

## Механизм

### 1. Пикер наполняется ключом настроек `modelPicker`

```json
{ "modelPicker": {
    "replaceBuiltInOptions": true,
    "options": [ { "model": "…", "label": "…", "description": "…", "behavesAs": "…" } ] } }
```

- `model` берётся **дословно**; произвольные id со слэшами принимаются.
- Источники и приоритет: `["policySettings","flagSettings","userSettings"]`.
  `flagSettings` = `--settings`, то есть overlay, который wisp-deck уже пишет
  (`write_claude_launch_settings`, `lib/settings-json.sh:77`).
- Побеждает один источник целиком, без merge.

### 2. Роутер-прокси на loopback выбирает эндпоинт и креды

Claude Code шлёт выбранную строку в поле `model`. Роутер её разбирает,
подменяет `Authorization` и форвардит.

Грамматика id: `wisp/<source>/<model>`, где `<source>` — один сегмент:

| префикс | источник | креды |
|---|---|---|
| `acct.<dir>` | Claude-логин `<accounts_dir>/<dir>` (`acct.default` = Keychain-логин) | OAuth из Keychain |
| `cfg.<file>` | subscription-профиль `<configs_dir>/<file>.json` | `ANTHROPIC_AUTH_TOKEN` из профиля |

`<model>` может содержать слэши (Featherless: `TurboVadim/Qwen3.8-27B-OBLITERATED`),
поэтому режем только по **первым двум** сегментам.

### 3. Окно контекста задаётся суффиксом id, а не глобальным env

Замерено через `/context`:

| строка | окно |
|---|---|
| встроенные `claude-*` | родное (1M у Opus 5) |
| `wisp/…` + `behavesAs: claude-opus-5` | 200k — `behavesAs` окно НЕ переносит |
| `wisp/…[1m]` | 1m — маркер срабатывает на сырой строке |

Роутер срезает `[1m]` перед форвардом и добавляет beta `context-1m-2025-08-07`.
Модели с реальным окном меньше 200k в список не попадают.

**Отменено при финальном ревью (2026-09-07): ростер `[1m]` больше не пишет.**
Окно 1M выдаётся всей сессии по сырой строке модели, и ничто не сужает его
обратно, когда пользователь в том же диалоге выбирает 200k-строку: транскрипт
уже больше лимита нового эндпоинта, а `/compact` шлёт тот же транскрипт плюс
промпт суммаризации, то есть больше уже упавшего турна. Все строки теперь
ровно 200k — единственная форма, которую нельзя заклинить выбором. `strip1M`,
`Want1M` и beta-заголовок оставлены как есть: 1M вернётся за размерным гардом.
Подробности и гарды — в `internal/allin/CLAUDE.md`.

## Учётные данные

macOS Keychain, сервис `Claude Code-credentials-<sha256(configDir)[:8]>`;
Default-логин — просто `Claude Code-credentials`.
Проверено: `~/.config/wisp-deck/claude-accounts/personal` → `7646b36d`, совпало.
JSON в поле пароля: `claudeAiOauth.accessToken` / `refreshToken` / `expiresAt`.

## Измерения (2026-09-07)

Панель залогинена в Default (`authfp 7e3c78de`), выбрана строка
`wisp/personal/claude-opus-5`; роутер подменил токен на Personal (`authfp 1ff73b43`):

```
17:09:48 | acct=session  | authfp=7e3c78de | 200   <- Default
17:16:51 | acct=personal | authfp=1ff73b43 | 200   <- Personal, ТА ЖЕ панель
17:17:43 | acct=personal | authfp=1ff73b43 | 200
```

Непрерывность диалога: на вопрос «какие два слова ты отвечал раньше» Personal
ответил `baseline crossaccount` — увидел реплику, сгенерированную под Default.
Побочно: Default отдал 429 при живом Personal.

Прочее, измеренное там же:
- `/context` даёт 156 запросов на `/v1/messages/count_tokens` против 4 на `/v1/messages`.
- Прогревочный запрос идёт на модель по умолчанию (`max_tokens:1`, `tools:0`), не на выбранную.
- OAuth-турн несёт beta `oauth-2025-04-20`; турн на API-ключе — нет.
- `Engine.Execute` (`internal/gptbridge/engine.go:140`) валидирует `model` по allowlist
  и отвечает `model %q is not available` — id для GPT надо переписывать в голый codex-id.

## Решения по v1

- **Refresh токенов не делаем.** На 401 роутер отдаёт понятную ошибку с именем аккаунта.
- **Объём:** сразу и Claude-логины, и все API-провайдеры.
- **Автофейловер на 429 не делаем** — 429 честно уходит наверх.

## Тупиковые пути

`CLAUDE_CODE_MODEL_CATALOG_URL` подписан вшитыми trusted roots и гейтится на
first-party/org. `CLAUDE_CODE_MODEL_CATALOG` — выключатель.
`ANTHROPIC_CUSTOM_MODEL_OPTION` добавляет ровно одну строку.

## Открытые риски

- Протухший токен неиспользуемого аккаунта лечится только ручным открытием этого логина.
- Одновременное использование двух подписок из одной сессии — вопрос к ToS Anthropic;
  решение за владельцем аккаунтов.
- `Enter` в пикере сохраняет выбор дефолтом для НОВЫХ сессий: записанный туда
  `wisp/…` id сломает запуск `claude` без роутера. v1 не мешает этому — только документирует.
