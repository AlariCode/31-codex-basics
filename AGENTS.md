# Repository Guidelines

## Repository Layout

The repository has two independently runnable applications:

- `frontend/` is the browser application.
- `backend/` is the service application.

Work in the appropriate directory and follow its local `AGENTS.md`; local instructions take precedence for files beneath that directory. Keep UI concerns in the frontend and server-side concerns in the backend. When a change crosses this boundary, document the API or configuration contract affected.

## Shared Development Practices

Keep changes focused and avoid editing generated output or dependency lockfiles unless the change requires them. Before submitting, run the relevant checks specified by the application guide. If a change touches both applications, verify both sides of the contract and state the commands run.

## Commits and Pull Requests

The available Git history has no established commit convention. Use concise imperative commit subjects, for example `Add uptime status endpoint`, and keep commits limited to one logical change.

Pull requests should describe the behavior change, verification performed, and any configuration or API-contract changes. Link related issues when available. Include screenshots for visible frontend changes and clearly note any required environment variables, migrations, or deployment steps.

## Коммиты

- Формат: Conventional Commits (feat, fix, refactor, test, docs, chore)
- Заголовок до 72 символов, в повелительном наклонении
- Без эмодзи, без «significantly improved» и прочей воды
- Тело — только если нужно объяснить «почему», а не «что»
- Один логический шаг — один коммит
