This is the browser application for Uptime, built with [Next.js](https://nextjs.org).

## Authentication setup

Create `frontend/.env.local` from `.env.example` and set the backend URL:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Start the backend with `CORS_ORIGIN=http://localhost:3005`. The frontend sends authentication requests with `credentials: "include"`: the backend owns the HttpOnly refresh cookie, while the access JWT exists only in React memory and is refreshed after a page reload.

Available pages:

- `/register` creates an account.
- `/login` signs in an existing user.
- `/` restores the browser session and displays its state.

The authenticated dashboard header contains a profile menu. Its logout action calls `POST /api/v1/auth/logout`, clears the HttpOnly refresh cookie on the backend, and returns the user to `/login`.

## Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3005](http://localhost:3005) with your browser to see the result.

You can start editing the page by modifying `app/page.tsx`. The page auto-updates as you edit the file.

This project uses [`next/font`](https://nextjs.org/docs/app/building-your-application/optimizing/fonts) to automatically optimize and load [Geist](https://vercel.com/font), a new font family for Vercel.

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js) - your feedback and contributions are welcome!

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/app/building-your-application/deploying) for more details.

## Monitor dashboard

The authenticated `/` page displays each monitor's last result and availability chart.
The shared selector offers 1 hour, 24 hours (default), 7 days and 30 days. Times use
the browser's local timezone. Bars expose counters on hover/focus and support arrow
keys. Gray means no observations, green means all succeeded, red includes failures.
The displayed percentage is a ratio of completed checks, not elapsed uptime.

Statuses refresh every 5 seconds and charts every 60 seconds; polling pauses while
the tab is hidden and resumes immediately on return. Failed refreshes preserve the
last loaded data. Results older than the configured interval plus 15 seconds are
marked stale. A URL edit preserves history, including observations of its former URL.

Requires the backend monitoring API and migration `000006_add_monitor_results`.
