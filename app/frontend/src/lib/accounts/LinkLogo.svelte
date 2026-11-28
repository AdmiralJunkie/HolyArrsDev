<script lang="ts">
	// A small brand mark for a Plex, Jellyfin, Emby or Kodi account or server: before a linked account in My account >
	// Linked sign-ins, and before a server's name in Linked apps > Media servers (the same simple-icons marks and
	// colours the Settings home cards use). Anything else draws nothing (hasLinkLogo says which kinds have one).
	// label: set it where the mark stands in for the words (a screen reader reads it); unset, the mark is decoration.
	import plex from 'simple-icons/icons/plex.svg?raw';
	import jellyfin from 'simple-icons/icons/jellyfin.svg?raw';
	import emby from 'simple-icons/icons/emby.svg?raw';
	import kodi from 'simple-icons/icons/kodi.svg?raw';

	let { kind, size = 14, label = '' }: { kind: string; size?: number; label?: string } = $props();

	const d = (svg: string) => / d="([^"]+)"/.exec(svg)?.[1] ?? '';
	const MARKS: Record<string, { path: string; color: string }> = {
		plex: { path: d(plex), color: '#EBAF00' },
		jellyfin: { path: d(jellyfin), color: '#00A4DC' },
		emby: { path: d(emby), color: '#52B54B' },
		kodi: { path: d(kodi), color: '#17B2E7' }
	};
	const mark = $derived(MARKS[kind]);
</script>

{#if mark}
	{#if label}
		<svg class="link-logo" viewBox="0 0 24 24" width={size} height={size} role="img" aria-label={label} focusable="false" data-testid="link-logo-{kind}"><title>{label}</title><path d={mark.path} fill={mark.color} /></svg>
	{:else}
		<svg class="link-logo" viewBox="0 0 24 24" width={size} height={size} aria-hidden="true" focusable="false" data-testid="link-logo-{kind}"><path d={mark.path} fill={mark.color} /></svg>
	{/if}
{/if}

<style>
	.link-logo { flex: none; display: inline-block; vertical-align: -2px; margin-inline-end: 6px; }
</style>
