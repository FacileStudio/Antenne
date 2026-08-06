export const TAG_TONE_COUNT = 12;

function hash(tag: string): number {
	let value = 0;
	for (const character of tag) {
		value = (value * 31 + character.charCodeAt(0)) >>> 0;
	}
	return value;
}

function circularDistance(left: number, right: number, size: number): number {
	const distance = Math.abs(left - right);
	return Math.min(distance, size - distance);
}

/**
 * Assigns each tag a tone from the twelve-hue scale.
 *
 * A plain hash would collide as soon as two tags land on the same hue, and two
 * identically-coloured pills side by side read as one category. This spreads
 * them: each tag prefers its hash's hue but pays a heavy penalty for a tone
 * already in use, so a small tag set gets twelve distinct colours and a large
 * one degrades evenly instead of clumping.
 */
export function tagToneMap(tags: string[]): Record<string, number> {
	const unique = [...new Set(tags.map((tag) => tag.trim()).filter(Boolean))];
	const sorted = unique.sort((left, right) => hash(left) - hash(right) || left.localeCompare(right));

	const usage = Array.from({ length: TAG_TONE_COUNT }, () => 0);
	const tones: Record<string, number> = {};

	for (const tag of sorted) {
		const preferred = hash(tag) % TAG_TONE_COUNT;
		let bestTone = 0;
		let bestScore = Number.POSITIVE_INFINITY;

		for (let tone = 0; tone < TAG_TONE_COUNT; tone += 1) {
			const score = usage[tone] * 100 + circularDistance(preferred, tone, TAG_TONE_COUNT);
			if (score < bestScore) {
				bestScore = score;
				bestTone = tone;
			}
		}

		usage[bestTone] += 1;
		tones[tag] = bestTone;
	}

	return tones;
}

export function tagTone(tag: string, tones?: Record<string, number>): number {
	if (tones && tag in tones) return tones[tag];
	return hash(tag) % TAG_TONE_COUNT;
}
