export function sanitizeTags(tags: string[]) {
  return [...new Set(tags.map((tag) => tag.trim()).filter(Boolean))];
}

export const TAG_TONE_COUNT = 12;

function getTagHash(tag: string) {
  let hash = 0;

  for (const character of tag) {
    hash = (hash * 31 + character.charCodeAt(0)) >>> 0;
  }

  return hash;
}

function getCircularDistance(left: number, right: number, size: number) {
  const distance = Math.abs(left - right);
  return Math.min(distance, size - distance);
}

export function createTagToneMap(tags: string[]) {
  const uniqueTags = sanitizeTags(tags);
  const sortedTags = [...uniqueTags].sort((left, right) => {
    const hashDifference = getTagHash(left) - getTagHash(right);
    return hashDifference !== 0 ? hashDifference : left.localeCompare(right);
  });
  const usageCounts = Array.from({ length: TAG_TONE_COUNT }, () => 0);
  const toneMap: Record<string, number> = {};

  for (const tag of sortedTags) {
    const preferredTone = getTagHash(tag) % TAG_TONE_COUNT;
    let bestTone = 0;
    let bestScore = Number.POSITIVE_INFINITY;

    for (let tone = 0; tone < TAG_TONE_COUNT; tone += 1) {
      const score = usageCounts[tone]! * 100 + getCircularDistance(preferredTone, tone, TAG_TONE_COUNT);

      if (score < bestScore) {
        bestScore = score;
        bestTone = tone;
      }
    }

    usageCounts[bestTone] += 1;
    toneMap[tag] = bestTone;
  }

  return toneMap;
}

export function getTagTone(tag: string, toneMap?: Record<string, number>) {
  if (toneMap && tag in toneMap) {
    return toneMap[tag]!;
  }

  return getTagHash(tag) % TAG_TONE_COUNT;
}
