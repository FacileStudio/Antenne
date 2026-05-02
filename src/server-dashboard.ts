const dashboardDistDir = new URL("../dashboard/.output/public/", import.meta.url);

function buildFallbackDashboardHtml() {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Nook Dashboard</title>
    <style>
      :root {
        color-scheme: dark;
        font-family: "Satoshi", "Helvetica Neue", Helvetica, Arial, sans-serif;
        background: #050505;
        color: #fff;
      }
      body {
        margin: 0;
        min-height: 100vh;
        display: grid;
        place-items: center;
        background: #050505;
        letter-spacing: -0.02em;
      }
      main {
        width: min(42rem, calc(100vw - 2rem));
        padding: 2rem;
        border: 1px solid rgb(255 255 255 / 0.1);
        border-radius: 1.5rem;
        background: rgb(255 255 255 / 0.03);
        box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.02);
      }
      h1 {
        margin-top: 0;
        font-size: 2rem;
      }
      p {
        color: rgb(255 255 255 / 0.58);
        line-height: 1.5;
      }
      code {
        padding: 0.2rem 0.45rem;
        border-radius: 999px;
        background: rgb(255 255 255 / 0.05);
      }
    </style>
  </head>
  <body>
    <main>
      <h1>Nook dashboard not built</h1>
      <p>Run <code>bun run dashboard:build</code> before serving the dashboard. Backend, monitors, and webhooks still work.</p>
    </main>
  </body>
</html>`;
}

export async function serveDashboardAsset(pathname: string, dashboardBuilt: boolean): Promise<Response> {
  if (!dashboardBuilt) {
    return new Response(buildFallbackDashboardHtml(), {
      headers: { "Content-Type": "text/html; charset=utf-8" },
    });
  }

  const relativePath = pathname === "/dashboard" || pathname === "/dashboard/"
    ? "index.html"
    : pathname.replace(/^\/dashboard\/?/, "");
  const file = Bun.file(new URL(relativePath || "index.html", dashboardDistDir));

  if (relativePath !== "index.html" && !(await file.exists())) {
    if (isDashboardAssetPath(pathname)) {
      return new Response("not found", { status: 404 });
    }

    return serveDashboardAsset("/dashboard/index.html", dashboardBuilt);
  }

  return new Response(file, {
    headers: relativePath === "index.html"
      ? {
        "Cache-Control": "private, no-store",
        Vary: "Cookie",
      }
      : {
        "Cache-Control": "public, max-age=3600",
      },
  });
}

export function isDashboardAssetPath(pathname: string) {
  return pathname.startsWith("/dashboard/_nuxt/")
    || /\.(?:css|js|mjs|png|webp|jpg|jpeg|gif|svg|ico|woff|woff2|json)$/i.test(pathname);
}
