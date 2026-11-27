// The owner's account picker in Settings > Users: an account linked to someone else shows whose it is and cannot be picked.
export type ServerAccount = { id: string; name: string; taken_by?: number; taken_name?: string };

export function accountChoice(a: ServerAccount, userId: number, t: (key: string, vars?: Record<string, string | number>) => string): { label: string; taken: boolean } {
	if (!a.taken_by || a.taken_by === userId) return { label: a.name, taken: false };
	return { label: `${a.name} · ${t('settings.users.accountLinked', { name: a.taken_name ?? '' })}`, taken: true };
}

// My account > Linked sign-ins: the owner's Jellyfin and Emby servers I may link, and my links to them.
export type MediaServer = { id: number; name: string; kind: 'jellyfin' | 'emby' };
export type MediaLink = { provider: string; name: string; kept?: boolean };
export const mediaKindWord = (k: string) => (k === 'emby' ? 'Emby' : 'Jellyfin');
