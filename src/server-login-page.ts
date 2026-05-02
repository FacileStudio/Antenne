import type { DashboardSessionConfig } from "./types";

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

export function renderLoginPage(session: DashboardSessionConfig, errorMessage?: string | null) {
  const error = errorMessage ? `<p id="error">${escapeHtml(errorMessage)}</p>` : `<p id="error" hidden>Wrong password.</p>`;

  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Nook Login</title>
    <link rel="icon" type="image/webp" href="/dashboard/nook-avatar.webp">
    <style>
      :root {
        color-scheme: dark;
        font-family: "Satoshi", "Helvetica Neue", Helvetica, Arial, sans-serif;
        background: #050505;
        color: #fff;
      }
      *, *::before, *::after { box-sizing: border-box; }
      body {
        margin: 0;
        min-height: 100vh;
        display: grid;
        place-items: center;
        background: #050505;
        letter-spacing: -0.02em;
        padding: 1rem;
      }
      main {
        width: min(24rem, 100%);
        display: grid;
        gap: 1.5rem;
        justify-items: stretch;
        text-align: left;
      }
      .brand {
        display: flex;
        align-items: center;
        gap: 1rem;
        justify-self: start;
      }
      .brand img {
        width: 3.5rem;
        height: 3.5rem;
        border-radius: 50%;
        object-fit: cover;
        border: 2px solid rgb(255 255 255 / 0.1);
      }
      .brand span {
        font-family: "Poppins", "Helvetica Neue", Helvetica, Arial, sans-serif;
        font-weight: 700;
        font-size: 2.4rem;
        letter-spacing: -0.04em;
        line-height: 1;
      }
      p {
        margin: 0;
        color: rgb(255 255 255 / 0.5);
        line-height: 1.55;
        font-size: 0.95rem;
      }
      form {
        width: 100%;
        display: grid;
        gap: 0.75rem;
      }
      label {
        display: block;
        text-align: left;
        color: rgb(255 255 255 / 0.58);
        font-size: 0.92rem;
      }
      input {
        display: block;
        width: 100%;
        font: inherit;
        padding: 0.9rem 1.25rem;
        border-radius: 1.4rem;
        border: 1px solid rgb(255 255 255 / 0.1);
        background: rgb(255 255 255 / 0.03);
        color: inherit;
        outline: none;
        transition: border-color 120ms ease, background 120ms ease;
      }
      input:focus {
        border-color: rgb(255 255 255 / 0.24);
        background: rgb(255 255 255 / 0.05);
      }
      button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
        width: 100%;
        font: inherit;
        padding: 0.9rem 1rem;
        border: 1px solid rgb(255 255 255 / 0.1);
        border-radius: 999px;
        background: #fff;
        color: #050505;
        font-weight: 700;
        cursor: pointer;
        transition: background 120ms ease, border-color 120ms ease;
      }
      button:hover { background: rgb(255 255 255 / 0.86); }
      button svg { flex: none; }
      #error {
        color: rgb(255 131 131);
        font-size: 0.9rem;
      }
    </style>
  </head>
  <body>
    <main>
      <div class="brand">
        <img src="/dashboard/nook-avatar.webp" alt="Nook">
        <span>Nook</span>
      </div>
      <p>Sign in as <b>${escapeHtml(session.username)}</b></p>
      <form action="/dashboard/login" method="post">
        <label for="password">Password</label>
        <input id="password" name="password" type="password" autocomplete="current-password" autofocus>
        <button type="submit">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>
          Log in
        </button>
      </form>
      ${error}
    </main>
  </body>
</html>`;
}
