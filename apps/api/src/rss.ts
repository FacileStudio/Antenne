import { formatTimestamp } from "./format";
import type { AlertContext, AppSettings } from "./types";
import { isRssProviderSettings, type RssProviderSettings } from "./webhooks/types";

interface RssItem {
  guid: string;
  title: string | null;
  link: string | null;
  description: string | null;
  pubDate: string | null;
  author: string | null;
}

interface RssFeed {
  title: string | null;
  link: string | null;
  items: RssItem[];
}

interface RssEntryState {
  timerId: Timer;
  running: boolean;
  seenGuids: Set<string>;
  initialized: boolean;
}

function extractTag(xml: string, tag: string): string | null {
  const pattern = new RegExp(`<${tag}[^>]*>([\\s\\S]*?)<\\/${tag}>`, "i");
  const match = xml.match(pattern);
  if (!match?.[1]) return null;
  return decodeEntities(match[1].trim());
}

function decodeEntities(value: string): string {
  return value
    .replace(/<!\[CDATA\[([\s\S]*?)\]\]>/g, "$1")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&amp;/g, "&")
    .replace(/&quot;/g, "\"")
    .replace(/&apos;/g, "'")
    .replace(/&#(\d+);/g, (_m, code: string) => String.fromCharCode(Number(code)))
    .replace(/&#x([0-9a-fA-F]+);/g, (_m, code: string) => String.fromCharCode(parseInt(code, 16)));
}

function parseRssItems(xml: string): RssItem[] {
  const items: RssItem[] = [];
  const itemPattern = /<item[\s>]([\s\S]*?)<\/item>/gi;
  let match: RegExpExecArray | null;

  while ((match = itemPattern.exec(xml)) !== null) {
    const block = match[1] ?? "";
    const guid = extractTag(block, "guid") ?? extractTag(block, "link") ?? `rss-${items.length}`;
    items.push({
      guid,
      title: extractTag(block, "title"),
      link: extractTag(block, "link"),
      description: extractTag(block, "description"),
      pubDate: extractTag(block, "pubDate"),
      author: extractTag(block, "author") ?? extractTag(block, "dc:creator"),
    });
  }

  return items;
}

function parseAtomItems(xml: string): RssItem[] {
  const items: RssItem[] = [];
  const entryPattern = /<entry[\s>]([\s\S]*?)<\/entry>/gi;
  let match: RegExpExecArray | null;

  while ((match = entryPattern.exec(xml)) !== null) {
    const block = match[1] ?? "";
    const linkMatch = block.match(/<link[^>]*href=["']([^"']*)["'][^>]*\/?>/i);
    const link = linkMatch?.[1] ? decodeEntities(linkMatch[1]) : null;
    const guid = extractTag(block, "id") ?? link ?? `atom-${items.length}`;

    items.push({
      guid,
      title: extractTag(block, "title"),
      link,
      description: extractTag(block, "summary") ?? extractTag(block, "content"),
      pubDate: extractTag(block, "published") ?? extractTag(block, "updated"),
      author: extractTag(block, "name"),
    });
  }

  return items;
}

function parseFeed(xml: string): RssFeed {
  const isAtom = /<feed[\s>]/i.test(xml);

  if (isAtom) {
    const feedLinkMatch = xml.match(/<link[^>]*rel=["']alternate["'][^>]*href=["']([^"']*)["'][^>]*\/?>/i)
      ?? xml.match(/<link[^>]*href=["']([^"']*)["'][^>]*\/?>/i);
    return {
      title: extractTag(xml, "title"),
      link: feedLinkMatch?.[1] ? decodeEntities(feedLinkMatch[1]) : null,
      items: parseAtomItems(xml),
    };
  }

  return {
    title: extractTag(xml.match(/<channel[\s>]([\s\S]*?)<\/channel>/i)?.[1] ?? xml, "title"),
    link: extractTag(xml.match(/<channel[\s>]([\s\S]*?)<\/channel>/i)?.[1] ?? xml, "link"),
    items: parseRssItems(xml),
  };
}

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function stripHtml(value: string): string {
  return value.replace(/<[^>]*>/g, "").trim();
}

function buildMessage(provider: RssProviderSettings, feed: RssFeed, item: RssItem): string {
  const parts = [
    `📡 <b>[RSS] ${escapeHtml(provider.name)}</b> new item`,
  ];

  if (item.title) {
    if (item.link) {
      parts.push(`<b><a href="${escapeHtml(item.link)}">${escapeHtml(item.title)}</a></b>`);
    } else {
      parts.push(`<b>${escapeHtml(item.title)}</b>`);
    }
  }

  if (item.description) {
    const plain = stripHtml(item.description);
    const snippet = plain.length > 300 ? `${plain.slice(0, 300)}…` : plain;
    parts.push(escapeHtml(snippet));
  }

  if (item.pubDate) {
    parts.push(`<code>${escapeHtml(formatTimestamp(item.pubDate))}</code>`);
  }

  if (feed.title) {
    parts.push(`Feed: ${escapeHtml(feed.title)}`);
  }

  return parts.join("<br/>");
}

export interface RssAlertContext extends AlertContext {
  providerType: "rss";
}

export function createRssManager(sendAlert: (message: string, context: RssAlertContext) => Promise<void>) {
  const entries = new Map<string, RssEntryState>();

  async function pollProvider(provider: RssProviderSettings) {
    const entry = entries.get(provider.id);
    if (!entry || entry.running || !provider.feedUrl) {
      return;
    }

    entry.running = true;

    try {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 30_000);

      let response: Response;
      try {
        response = await fetch(provider.feedUrl, {
          signal: controller.signal,
          headers: { "User-Agent": "Nook/1.0 rss-bot" },
        });
      } finally {
        clearTimeout(timeout);
      }

      if (!response.ok) {
        console.error(`[rss] ${provider.name}: HTTP ${response.status}`);
        return;
      }

      const xml = await response.text();
      const feed = parseFeed(xml);

      if (!entry.initialized) {
        for (const item of feed.items) {
          entry.seenGuids.add(item.guid);
        }
        entry.initialized = true;
        console.log(`[rss] initialized ${provider.name} with ${feed.items.length} existing items`);
        return;
      }

      const newItems = feed.items.filter((item) => !entry.seenGuids.has(item.guid));
      for (const item of newItems) {
        entry.seenGuids.add(item.guid);
        await sendAlert(buildMessage(provider, feed, item), {
          providerId: provider.id,
          providerName: provider.name,
          providerType: "rss",
          event: `new item: ${item.title ?? "untitled"}`,
          eventType: "rss.item.new",
          eventTitle: item.title ?? "New feed item",
          eventUrl: item.link,
          occurredAt: item.pubDate,
          actor: item.author ? { name: item.author } : null,
          subject: {
            type: "unknown",
            id: item.guid,
            name: item.title ?? "Feed item",
            url: item.link,
          },
          normalizedPayload: {
            guid: item.guid,
            title: item.title,
            link: item.link,
            description: item.description ? stripHtml(item.description).slice(0, 500) : null,
            pubDate: item.pubDate,
            author: item.author,
            feedTitle: feed.title,
            feedUrl: provider.feedUrl,
          },
        });
      }

      if (entry.seenGuids.size > 5000) {
        const guidsArray = [...entry.seenGuids];
        entry.seenGuids = new Set(guidsArray.slice(guidsArray.length - 2000));
      }
    } catch (error) {
      console.error(`[rss] ${provider.name}: ${error instanceof Error ? error.message : String(error)}`);
    } finally {
      entry.running = false;
    }
  }

  function stopAll() {
    for (const entry of entries.values()) {
      clearInterval(entry.timerId);
    }
    entries.clear();
  }

  function apply(settings: AppSettings) {
    stopAll();

    for (const provider of settings.providers.items.filter(isRssProviderSettings).filter((p) => p.enabled)) {
      const timerId = setInterval(() => void pollProvider(provider), provider.pollInterval * 1000);
      entries.set(provider.id, {
        timerId,
        running: false,
        seenGuids: new Set(),
        initialized: false,
      });
      console.log(`[rss] polling ${provider.name} (${provider.feedUrl}) every ${provider.pollInterval}s`);
      void pollProvider(provider);
    }
  }

  return {
    apply,
    stopAll,
  };
}
