// The owner's account picker in Settings > Users: an account linked to someone else shows whose it is and cannot be picked.
export type ServerAccount = { id: string; name: string; taken_by?: number; taken_name?: string };

export function accountChoice(a: ServerAccount, userId: number, t: (key: string, vars?: Record<string, string | number>) => string): { label: string; taken: boolean } {
	if (!a.taken_by || a.taken_by === userId) return { label: a.name, taken: false };
	return { label: `${a.name} · ${t('settings.users.accountLinked', { name: a.taken_name ?? '' })}`, taken: true };
}

// My account > Linked sign-ins: the owner's Jellyfin and Emby servers I may link, and my links to them.
export type MediaServer = { id: number; name: string; kind: 'jellyfin' | 'emby'; offline?: boolean };
export type MediaLink = { provider: string; name: string; kept?: boolean };
export const mediaKindWord = (k: string) => (k === 'emby' ? 'Emby' : 'Jellyfin');

// A server's choice in the Link menu: "Jellyfin · Den", or just the name when it already says the kind ("Jellyfin", "My Jellyfin").
export function mediaServerLabel(m: { name: string; kind: string }): string {
	const kind = mediaKindWord(m.kind);
	const name = m.name.trim();
	if (!name) return kind;
	return name.toLowerCase().includes(kind.toLowerCase()) ? name : `${kind} · ${name}`;
}

/**
 * What is wrong with one of my media server links (HolyArrsDev), or null:
 * gone: the owner has no server of that kind connected; offline: every one of that kind failed its last call;
 * reconnect: a Jellyfin server is back but my sign-in for playing went with the removed one (playerLost).
 */
export type LinkProblem = 'gone' | 'offline' | 'reconnect' | null;
export function linkProblem(l: MediaLink, servers: MediaServer[], playerLost: boolean): LinkProblem {
	const mine = servers.filter((s) => s.kind === l.provider);
	if (!mine.length) return 'gone';
	if (mine.every((s) => s.offline)) return 'offline';
	if (l.provider === 'jellyfin' && playerLost && !l.kept) return 'reconnect';
	return null;
}

/** How many of my media server links need attention (My account's page pill). */
export function accountIssues(links: MediaLink[], servers: MediaServer[], playerLost: boolean): number {
	return links.filter((l) => linkProblem(l, servers, playerLost)).length;
}

/** Kinds that LinkLogo draws a brand mark for (others keep their words, as a chip). */
export const hasLinkLogo = (kind: string) => kind === 'plex' || kind === 'jellyfin' || kind === 'emby' || kind === 'kodi';
