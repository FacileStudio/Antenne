import type { WebhookProviderDefinition } from "../types";

interface GitPushPayload {
  ref?: string;
  before?: string;
  after?: string;
  compare?: string;
  commits?: Array<{
    id?: string;
    message?: string;
    url?: string;
    author?: { name?: string; username?: string };
  }>;
  head_commit?: {
    id?: string;
    message?: string;
    url?: string;
    author?: { name?: string; username?: string };
  };
  pusher?: { name?: string };
  sender?: { login?: string; username?: string };
  repository?: { full_name?: string; name?: string; html_url?: string; url?: string };
  user_name?: string;
  project?: { path_with_namespace?: string; web_url?: string };
  total_commits_count?: number;
}

interface GitIssuePayload {
  action?: string;
  issue?: {
    title?: string;
    html_url?: string;
    url?: string;
    number?: number;
    id?: number;
  };
  pull_request?: {
    title?: string;
    html_url?: string;
    url?: string;
    number?: number;
    id?: number;
    merged?: boolean;
  };
  object_attributes?: {
    title?: string;
    url?: string;
    action?: string;
    state?: string;
    iid?: number;
  };
  repository?: { full_name?: string; name?: string; html_url?: string; url?: string };
  sender?: { login?: string; username?: string };
  user?: { name?: string; username?: string };
}

type GitPayload = GitPushPayload & GitIssuePayload;

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function getRepoName(payload: GitPayload): string {
  return payload.repository?.full_name
    ?? payload.project?.path_with_namespace
    ?? payload.repository?.name
    ?? "unknown";
}

function getActor(payload: GitPayload): string {
  return payload.pusher?.name
    ?? payload.sender?.login
    ?? payload.sender?.username
    ?? payload.user_name
    ?? payload.user?.name
    ?? payload.user?.username
    ?? "someone";
}

function shortRef(ref: string): string {
  return ref.replace(/^refs\/(heads|tags)\//, "");
}

function shortSha(sha: string): string {
  return sha.slice(0, 7);
}

function formatPush(payload: GitPayload): string | null {
  const repo = escapeHtml(getRepoName(payload));
  const actor = escapeHtml(getActor(payload));
  const branch = payload.ref ? escapeHtml(shortRef(payload.ref)) : "unknown";
  const commits = payload.commits ?? [];
  const count = commits.length || payload.total_commits_count || 0;

  const parts = [`<b>[${repo}]</b> ${actor} pushed ${count} commit${count !== 1 ? "s" : ""} to <code>${branch}</code>`];

  for (const commit of commits.slice(0, 3)) {
    const sha = commit.id ? `<code>${shortSha(commit.id)}</code>` : "";
    const msg = commit.message ? escapeHtml(commit.message.split("\n")[0] ?? "") : "no message";
    parts.push(`  ${sha} ${msg}`);
  }

  if (commits.length > 3) {
    parts.push(`  ... and ${commits.length - 3} more`);
  }

  if (payload.compare) {
    parts.push(`<a href="${escapeHtml(payload.compare)}">Compare</a>`);
  }

  return parts.join("<br/>");
}

function formatPullRequest(payload: GitPayload): string | null {
  const repo = escapeHtml(getRepoName(payload));
  const actor = escapeHtml(getActor(payload));
  const pr = payload.pull_request ?? payload.object_attributes;
  if (!pr) return null;

  const title = escapeHtml(pr.title ?? "untitled");
  const action = payload.action ?? payload.object_attributes?.action ?? "updated";
  const number = payload.pull_request?.number ?? payload.object_attributes?.iid ?? "";
  const url = payload.pull_request?.html_url ?? payload.object_attributes?.url;

  const label = url ? `<a href="${escapeHtml(url)}">#${number}</a>` : `#${number}`;

  return `<b>[${repo}]</b> ${actor} ${escapeHtml(action)} PR ${label}: ${title}`;
}

function formatIssue(payload: GitPayload): string | null {
  const repo = escapeHtml(getRepoName(payload));
  const actor = escapeHtml(getActor(payload));
  const issue = payload.issue ?? payload.object_attributes;
  if (!issue) return null;

  const title = escapeHtml(issue.title ?? "untitled");
  const action = payload.action ?? payload.object_attributes?.action ?? "updated";
  const number = payload.issue?.number ?? payload.object_attributes?.iid ?? "";
  const url = payload.issue?.html_url ?? payload.object_attributes?.url;

  const label = url ? `<a href="${escapeHtml(url)}">#${number}</a>` : `#${number}`;

  return `<b>[${repo}]</b> ${actor} ${escapeHtml(action)} issue ${label}: ${title}`;
}

function formatGenericEvent(payload: GitPayload): string | null {
  const repo = escapeHtml(getRepoName(payload));
  const actor = escapeHtml(getActor(payload));
  return `📡 <b>[${repo}]</b> ${actor} triggered a webhook event`;
}

function isPush(payload: GitPayload): boolean {
  return Boolean(payload.ref && (payload.commits || payload.total_commits_count !== undefined));
}

function isPullRequest(payload: GitPayload): boolean {
  return Boolean(payload.pull_request || (payload.object_attributes && !payload.issue));
}

function isIssue(payload: GitPayload): boolean {
  return Boolean(payload.issue);
}

function getPrimaryUrl(payload: GitPayload): string | null {
  return payload.pull_request?.html_url
    ?? payload.issue?.html_url
    ?? payload.object_attributes?.url
    ?? payload.compare
    ?? payload.repository?.html_url
    ?? payload.project?.web_url
    ?? null;
}

function formatGitAlert(payload: GitPayload): string | null {
  if (isPush(payload)) return formatPush(payload);
  if (isPullRequest(payload)) return formatPullRequest(payload);
  if (isIssue(payload)) return formatIssue(payload);
  return formatGenericEvent(payload);
}

function parsePayload(payload: unknown): GitPayload {
  return (payload ?? {}) as GitPayload;
}

export const gitWebhookProvider: WebhookProviderDefinition = {
  type: "git",
  label: "Git",
  defaultPath: "/webhook/git",
  defaultSecretHeader: "x-hub-signature-256",
  formatMessage(payload) {
    return formatGitAlert(parsePayload(payload));
  },
  describeEvent(payload) {
    const parsed = parsePayload(payload);
    const repo = getRepoName(parsed);
    if (isPush(parsed)) return `push to ${repo}`;
    if (isPullRequest(parsed)) return `PR on ${repo}`;
    if (isIssue(parsed)) return `issue on ${repo}`;
    return `event on ${repo}`;
  },
  buildEvent(payload) {
    const parsed = parsePayload(payload);
    const repo = getRepoName(parsed);
    const actor = getActor(parsed);
    const headCommit = parsed.head_commit ?? parsed.commits?.[0] ?? null;
    const pr = parsed.pull_request ?? parsed.object_attributes;
    const issue = parsed.issue;
    const eventType = isPush(parsed)
      ? "git.push"
      : isPullRequest(parsed)
        ? "git.pull_request"
        : isIssue(parsed)
          ? "git.issue"
          : "git.event";
    return {
      eventType,
      eventTitle: repo,
      eventUrl: getPrimaryUrl(parsed),
      actor: {
        name: actor,
      },
      subject: {
        type: "repository",
        id: repo,
        name: repo,
        url: parsed.repository?.html_url ?? parsed.project?.web_url ?? null,
      },
      normalizedPayload: {
        repository: repo,
        ref: parsed.ref ? shortRef(parsed.ref) : null,
        before: parsed.before ?? null,
        after: parsed.after ?? null,
        commitSha: headCommit?.id ?? parsed.after ?? null,
        commitMessage: headCommit?.message?.split("\n")[0] ?? null,
        commitCount: parsed.commits?.length ?? parsed.total_commits_count ?? null,
        pullRequestId: parsed.pull_request?.number ?? parsed.object_attributes?.iid ?? null,
        issueId: issue?.number ?? null,
        action: parsed.action ?? parsed.object_attributes?.action ?? null,
        merged: parsed.pull_request?.merged ?? null,
        url: getPrimaryUrl(parsed),
        status: eventType === "git.push"
          ? "pushed"
          : eventType === "git.pull_request"
            ? parsed.pull_request?.merged
              ? "merged"
              : (parsed.action ?? parsed.object_attributes?.action ?? null)
            : eventType === "git.issue"
              ? (parsed.action ?? parsed.object_attributes?.action ?? null)
              : null,
      },
    };
  },
};
