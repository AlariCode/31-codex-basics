# Repository Guidelines

## Repository Layout

The repository has two independently runnable applications:

- `frontend/` is the browser application.
- `backend/` is the service application.

Work in the appropriate directory and follow its local `AGENTS.md`; local instructions take precedence for files beneath that directory. Keep UI concerns in the frontend and server-side concerns in the backend. When a change crosses this boundary, document the API or configuration contract affected.

## Shared Development Practices

Keep changes focused and avoid editing generated output or dependency lockfiles unless the change requires them. Before submitting, run the relevant checks specified by the application guide. If a change touches both applications, verify both sides of the contract and state the commands run.

## GitHub Flow

- Before starting each task, create a separate branch from the current default branch; do not work directly in `main` or another shared branch.
- Name task branches using the type and a concise transliterated feature or fix name: `feat/<transliterated-name>` for new functionality and `fix/<transliterated-name>` for bug fixes.
- Use lowercase Latin characters, separate words with hyphens, and avoid spaces, punctuation, and Cyrillic characters in branch names.

## Commits and Pull Requests

The available Git history has no established commit convention. Use concise imperative commit subjects, for example `Add uptime status endpoint`, and keep commits limited to one logical change.

Pull requests should describe the behavior change, verification performed, and any configuration or API-contract changes. Link related issues when available. Include screenshots for visible frontend changes and clearly note any required environment variables, migrations, or deployment steps.

## Коммиты

- Формат: Conventional Commits (feat, fix, refactor, test, docs, chore)
- Заголовок до 72 символов, в повелительном наклонении
- Без эмодзи, без «significantly improved» и прочей воды
- Тело — только если нужно объяснить «почему», а не «что»
- Один логический шаг — один коммит

## Pull Request Workflow

- Создавай PR из текущей task-ветки в `main`; не создавай PR из `main` в `main`.
- Перед PR проверь `git status`, `git log main..HEAD`, `git diff --check` и убедись, что в PR не попадут незакоммиченные или нерелевантные файлы.
- Перед отправкой выполни проверки из локальных `AGENTS.md` для каждого затронутого приложения и укажи команды и результат в описании PR.
- Отправляй ветку командой `git push -u origin <branch-name>`. Если remote `origin` отсутствует, добавь его только после проверки целевого URL.
- Создавай PR через GitHub CLI: `gh pr create --base main --head <branch-name> --title "<conventional-commit-title>" --body-file <body-file>`.
- Название PR должно соответствовать Conventional Commits, быть коротким, в повелительном наклонении и не превышать 72 символа.
- Описание PR оформляй разделами `Summary`, `Changes`, `API / Configuration / Migrations`, `Verification`, `Risks / Follow-ups` и `Screenshots` при наличии UI-изменений.
- В `Summary` опиши пользовательский или системный эффект, в `Changes` — существенные изменения по приложениям, в `Verification` — фактически выполненные проверки.
- Явно указывай миграции, новые переменные окружения, изменения API-контрактов, ограничения и известные проблемы сборки.
- После создания сообщи URL PR, ветку-источник, ветку-назначение и оставшиеся локальные изменения.
