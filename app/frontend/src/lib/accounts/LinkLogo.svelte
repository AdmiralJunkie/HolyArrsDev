<script lang="ts">
	// A small brand mark before a linked Plex, Jellyfin or Emby account in My account > Linked sign-ins
	// (the same simple-icons marks and colours the Settings home cards use). Anything else draws nothing.
	import plex from 'simple-icons/icons/plex.svg?raw';
	import jellyfin from 'simple-icons/icons/jellyfin.svg?raw';
	import emby from 'simple-icons/icons/emby.svg?raw';

	let { kind }: { kind: string } = $props();

	const d = (svg: string) => / d="([^"]+)"/.exec(svg)?.[1] ?? '';
	const MARKS: Record<string, { path: string; color: string }> = {
		plex: { path: d(plex), color: '#EBAF00' },
		jellyfin: { path: d(jellyfin), color: '#00A4DC' },
		emby: { path: d(emby), color: '#52B54B' }
	};
	const mark = $derived(MARKS[kind]);
</script>

{#if mark}
	<svg class="link-logo" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true" focusable="false" data-testid="link-logo-{kind}"><path d={mark.path} fill={mark.color} /></svg>
{/if}

<style>
	.link-logo { flex: none; display: inline-block; vertical-align: -2px; margin-inline-end: 6px; }
</style>
