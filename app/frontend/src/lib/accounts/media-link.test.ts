import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { accountChoice } from './media-link.ts';

const mine = readFileSync(new URL('./MediaServerLink.svelte', import.meta.url), 'utf8');
const t = (key: string, vars?: Record<string, string | number>) => key + (vars ? ` ${vars.name}` : '');

const security = readFileSync(new URL('./AccountSecurity.svelte', import.meta.url), 'utf8');

describe('My account: Jellyfin and Emby links live under Linked sign-ins', () => {
	it('lists my media server links and the sign-in form inside the Linked sign-ins row', () => {
		const row = security.slice(security.indexOf('id="linked-me"'));
		expect(row.indexOf('<MediaServerLink bind:links={mediaLinks} bind:open={mediaOpen} />')).toBeGreaterThan(0);
		expect(security.match(/<MediaServerLink\b/g)).toHaveLength(1);
	});
	it('offers each connected server in the Link menu', () => {
		expect(security).toMatch(/\{#each mediaServers as m \(m\.id\)\}[\s\S]*?mediaOpen = m; linkOpen = false;[\s\S]*?mediaKindWord\(m\.kind\)\} · \{m\.name\}/);
	});
	it('shows the row when only a media server can be linked', () => {
		expect(security).toMatch(/\|\| mediaServers\.length \|\| mediaLinks\.length\}/);
	});
	it('has no separate Watched marks section any more', () => {
		expect(mine).not.toMatch(/accounts\.watch\.title/);
	});
});

describe('accountChoice', () => {
	it('offers a free account and the one this person already has', () => {
		expect(accountChoice({ id: 'a', name: 'Mum' }, 4, t)).toEqual({ label: 'Mum', taken: false });
		expect(accountChoice({ id: 'a', name: 'Mum', taken_by: 4, taken_name: 'ana' }, 4, t)).toEqual({ label: 'Mum', taken: false });
	});
	it('marks an account linked to someone else as taken, with their name', () => {
		expect(accountChoice({ id: 'k', name: 'Kid', taken_by: 9, taken_name: 'leo' }, 4, t)).toEqual({ label: 'Kid · settings.users.accountLinked leo', taken: true });
	});
});

describe('My account: keep a Jellyfin sign-in for the player (HolyArrsDev)', () => {
	it('offers keeping only on a Jellyfin server and sends it only there', () => {
		expect(mine).toMatch(/\{#if open\.kind === 'jellyfin'\}[\s\S]*data-testid="media-link-keep"/);
		expect(mine).toMatch(/if \(open\.kind === 'jellyfin'\) body\.keep_for_playback = keep;/);
	});
	it('says when a link keeps a sign-in for playing here', () => {
		expect(mine).toMatch(/\{#if l\.kept\} · \{t\('accounts\.watch\.keptForPlayback'\)\}\{\/if\}/);
	});
});
