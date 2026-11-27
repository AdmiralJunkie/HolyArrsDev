// HolyArrsDev: whether my Jellyfin player sign-in went with a server the owner removed. My account sets it when it
// loads my links (or after I link or unlink); the app frame reads it every few minutes, pins playerPins() in the bell
// and puts a red dot on Account while it is true.
import { api } from '#lib/core/api.ts';
import { t } from '#lib/core/i18n/i18n.svelte.ts';
import { accountIssues, type MediaLink, type MediaServer } from './media-link.ts';

/** lost: my player sign-in went with a removed server; issues: my media links that need attention (My account's pill). */
export const playerStatus = $state({ lost: false, issues: 0 });

export async function loadPlayerStatus(): Promise<void> {
	const r = await api.get<{ servers?: MediaServer[]; links?: MediaLink[]; player_lost?: boolean }>('/auth/media-link').catch(() => null);
	if (!r) return;
	playerStatus.lost = !!r.player_lost;
	playerStatus.issues = accountIssues(r.links ?? [], r.servers ?? [], playerStatus.lost);
}

/** The pinned alert while it is lost (the alerts panel's PinnedAlert shape); none otherwise. */
export function playerPins(): { key: string; title: string; body: string; tone: 'warn'; href: string; label: string }[] {
	if (!playerStatus.lost) return [];
	return [{ key: 'player-lost', title: t('frame.feeds.playerLost'), body: t('frame.feeds.playerLostBody'), tone: 'warn', href: '/account', label: t('frame.layout.account') }];
}
