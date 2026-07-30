<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

# Frontend Guidelines

## Structure

The Next.js App Router code lives in `src/app/`; use framework route names such as `page.tsx` and `layout.tsx`. Put static assets in `public/`. Use the `@/` import alias for modules under `src/`.

## Commands

Run these from `frontend/`:

```bash
npm install       # install dependencies
npm run dev       # start the local server
npm run lint      # run ESLint and Next.js checks
npm run build     # create a production build
```

## Style and Testing

Write strict TypeScript. Use PascalCase for React component exports and retain the established App Router file names. Follow the repository ESLint configuration; do not edit `.next/` output.

There is no frontend test script yet. When adding behavior that needs automated coverage, add colocated `*.test.ts` or `*.test.tsx` files (or a documented test directory) and the corresponding npm script. Run lint and build before submitting.

## Review Notes

For visible changes, include screenshots in the pull request. Describe any frontend-to-backend API dependency or environment-variable change.
