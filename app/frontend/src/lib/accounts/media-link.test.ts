import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { accountChoice, mediaServerLabel } from './media-link.ts';

const mine = readFileSync(new URL('./MediaServerLink.svelte', import.meta.url), 'utf8');
const t = (key: string, vars?: Record<string, string | number>) => key + (vars ? ` ${vars.name}` : '');

const security = readFileSync(new URL('./AccountSecurity.svelte', import.meta.url), 'utf8');

describe('My account: Jellyfin and Emby links live under Linked sign-ins', () => {
	it('lists my media server links and the sign-in form inside the Linked sign-ins row', () => {
		const row = security.slice(security.indexOf('id="linked-me"'));
		expect(row.indexOf('<MediaServerLink servers={mediaServers} bind:links={mediaLinks} bind:open={mediaOpen} />')).toBeGreaterThan(0);
		expect(security.match(/<MediaServerLink\b/g)).toHaveLength(1);
	});
	it('offers each connected server in the Link menu', () => {
		expect(security).toMatch(/\{#each mediaServers as m \(m\.id\)\}[\s\S]*?mediaOpen = m; linkOpen = false;[\s\S]*?mediaServerLabel\(m\)\}/);
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
		expect(mine).toMatch(/\{:else if l\.kept\}\{` · \$\{t\('accounts\.watch\.keptForPlayback'\)\}`\}\{\/if\}/);
	});
});

describe('mediaServerLabel', () => {
	it('names the kind and the server', () => {
		expect(mediaServerLabel({ name: 'Den', kind: 'jellyfin' })).toBe('Jellyfin · Den');
		expect(mediaServerLabel({ name: 'Den', kind: 'emby' })).toBe('Emby · Den');
	});
	it('does not say the kind twice', () => {
		expect(mediaServerLabel({ name: 'Jellyfin', kind: 'jellyfin' })).toBe('Jellyfin');
		expect(mediaServerLabel({ name: 'my jellyfin', kind: 'jellyfin' })).toBe('my jellyfin');
		expect(mediaServerLabel({ name: '  ', kind: 'jellyfin' })).toBe('Jellyfin');
	});
});

describe('Linked sign-ins: logos and unlinking', () => {
	it('puts the brand mark before Plex and media server links', () => {
		expect(mine).toMatch(/<span class="small"><LinkLogo kind=\{l\.provider\} \/>/);
		expect(security).toMatch(/<span class="small"><LinkLogo kind="plex" \/>/);
	});
	it('confirms every unlink with the red button', () => {
		expect(mine).toMatch(/variant="destructive" type="button" onclick=\{\(\) => unlink\(l\.provider\)\}/);
		expect(security).toMatch(/variant="destructive" type="button" onclick=\{unlinkPlexMine\}/);
		expect(security).toMatch(/variant="destructive" type="button" onclick=\{\(\) => unlink\(l\)\}/);
	});
});

describe('Linked sign-ins without a Jellyfin server (HolyArrsDev)', () => {
	it('shows Jellyfin greyed out, with why, when no Jellyfin server is connected', () => {
		expect(security).toMatch(/\{#if !jellyfinUp\}[\s\S]*?title=\{t\('accounts\.watch\.noServer'\)\}[\s\S]*?<Button type="button" disabled[^>]*data-testid="media-link-none">Jellyfin<\/Button>/);
		expect(security).toMatch(/\.no-server \{[^}]*cursor: not-allowed/);
	});
	it('greys a link whose server is gone and says so instead of the playback note', () => {
		expect(mine).toMatch(/<li class:gone=\{!connected\(l\.provider\)\}>/);
		expect(mine).toMatch(/\{#if !connected\(l\.provider\)\}\{` · \$\{t\('accounts\.watch\.serverGone'\)\}`\}\{:else if l\.kept\}/);
	});
	it('clears the lost alert when I link Jellyfin again or unlink it', () => {
		expect(mine).toMatch(/if \(l\.provider === 'jellyfin'\) playerStatus\.lost = false;/);
		expect(mine).toMatch(/if \(kind === 'jellyfin'\) playerStatus\.lost = false;/);
	});
});

describe('playerPins (HolyArrsDev)', () => {
	it('pins one warning that opens My account only while the player sign-in is lost', async () => {
		const { playerPins, playerStatus } = await import('./player-status.svelte.ts');
		playerStatus.lost = false;
		expect(playerPins()).toEqual([]);
		playerStatus.lost = true;
		const p = playerPins();
		expect(p).toHaveLength(1);
		expect(p[0]).toMatchObject({ key: 'player-lost', tone: 'warn', href: '/account' });
		playerStatus.lost = false;
	});
});

describe('Account menu dot (HolyArrsDev)', () => {
	const menu = readFileSync(new URL('../ui/Menu.svelte', import.meta.url), 'utf8');
	const layout = readFileSync(new URL('../../routes/+layout.svelte', import.meta.url), 'utf8');
	it('menu items can carry the red dot', () => {
		expect(menu.match(/\{#if item\.dot\}<span class="flag"/g)).toHaveLength(2);
		expect(menu).toMatch(/\.flag \{[^}]*background: var\(--fail\)/);
	});
	it('lights it on Account while the player sign-in is lost, and counts the pin on the bell', () => {
		expect(layout).toMatch(/href: to\('\/account'\), dot: playerStatus\.lost/);
		expect(layout).toMatch(/settingsPages, \[\.\.\.playerPins\(\),/);
	});
});
