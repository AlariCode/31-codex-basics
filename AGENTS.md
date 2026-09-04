# Repository Guidelines

## Repository Layout

The repository has two independently runnable applications:

- `frontend/` is the browser application.
- `backend/` is the service application.

## When starting a new task use GitHub Flow

Details in file: docs/rules/github-flow.md

## When creating Pull Request

Use rules in file: docs/rules/pr.md

## Browser work and verification

- For browser interaction in this project, use Playwright MCP.
- After implementing a task that changes browser-visible behaviour or layout, invoke the `playwright-mcp-qa` skill before reporting completion. It requires a functional and visual verification in Playwright MCP.

## Комментарии в коде

- Комментируй ПОЧЕМУ, а не что, так как это видно из кода
- Очевидное не комментируй, лучше используй правильные наименования
- Пубуличнный функции doc comment
- Сложную арифметику, поясняй радом
- Меняешь код, актуализируй коммнетарий
