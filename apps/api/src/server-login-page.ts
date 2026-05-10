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
      .brand svg {
        width: 3.5rem;
        height: 3.5rem;
        flex: none;
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
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true">
          <path fill="#1CDB81" d="M10.31 17.344c.767-.876 1.151-1.314 1.625-1.342q.065-.004.13 0c.474.028.858.466 1.625 1.342c1.67 1.906 2.505 2.858 2.271 3.68q-.03.107-.074.206C15.543 22 14.362 22 12 22s-3.543 0-3.887-.77a2 2 0 0 1-.074-.206c-.234-.822.6-1.774 2.27-3.68M14.5 12.5a2.5 2.5 0 1 1-5 0a2.5 2.5 0 0 1 5 0"/>
          <path fill="#1CDB81" fill-rule="evenodd" d="M12 8.035c-2.697 0-4.884 2.151-4.884 4.806a4.75 4.75 0 0 0 1.43 3.398a.68.68 0 0 1 0 .97a.706.706 0 0 1-.986 0a6.1 6.1 0 0 1-1.84-4.368c0-3.413 2.812-6.18 6.28-6.18s6.279 2.767 6.279 6.18a6.1 6.1 0 0 1-1.84 4.369a.706.706 0 0 1-.986 0a.68.68 0 0 1 0-.971a4.75 4.75 0 0 0 1.43-3.398c0-2.655-2.186-4.806-4.883-4.806" clip-rule="evenodd" opacity=".7"/>
          <path fill="#1CDB81" fill-rule="evenodd" d="M12 4.373c-4.752 0-8.605 3.791-8.605 8.468c0 2.338.963 4.454 2.52 5.987a.68.68 0 0 1 0 .97a.706.706 0 0 1-.986 0A9.73 9.73 0 0 1 2 12.842C2 7.406 6.477 3 12 3s10 4.406 10 9.84a9.73 9.73 0 0 1-2.929 6.959a.706.706 0 0 1-.987 0a.68.68 0 0 1 0-.971a8.37 8.37 0 0 0 2.52-5.987c0-4.677-3.852-8.468-8.604-8.468" clip-rule="evenodd" opacity=".4"/>
        </svg>
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
