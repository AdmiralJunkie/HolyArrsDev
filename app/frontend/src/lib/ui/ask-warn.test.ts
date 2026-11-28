import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { answer, ask, asking } from './ask.svelte.ts';

// HolyArrsDev: a confirm can carry a red line saying what else it will do.
describe('ask warn', () => {
	it('passes the warning to the sheet and still answers', async () => {
		const p = ask('Remove Den?', { yes: 'Remove', danger: true, warn: '2 people have linked' });
		expect(asking.now?.warn).toBe('2 people have linked');
		answer(true);
		expect(await p).toBe(true);
	});
	it('shows it in red under the question', () => {
		const host = readFileSync(new URL('../../routes/(frame)/AskHost.svelte', import.meta.url), 'utf8');
		expect(host).toMatch(/\{#if warn\}<p class="ask-warn"/);
		expect(host).toMatch(/\.ask-warn \{[^}]*color: var\(--fail-text, var\(--fail\)\)/);
	});
	it('the media server Remove warns how many people linked the last Jellyfin server', () => {
		const page = readFileSync(new URL('../../routes/settings/media-servers/+page.svelte', import.meta.url), 'utf8');
		expect(page).toMatch(/m\.kind !== 'jellyfin' \|\| servers\.some\(\(s\) => s\.id !== m\.id && s\.kind === 'jellyfin' && s\.enabled\)/);
		expect(page).toMatch(/danger: true, warn \}/);
	});
});
