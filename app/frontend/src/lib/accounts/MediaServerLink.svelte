<script lang="ts">
	// My account > Linked sign-ins: my links to the owner's Jellyfin or Emby servers (so the Library shows what I watched),
	// and the name-and-password form for the server picked from the Link menu (open).
	// The password goes to that server only; HolyArrs keeps just the account's id there, unless the person asks
	// (Jellyfin only, HolyArrsDev fork) to keep the server's sign-in for the built-in player.
	import { t } from '#lib/core/i18n/i18n.svelte.ts';
	import Button from '#lib/ui/Button.svelte';
	import Confirm from '#lib/ui/Confirm.svelte';
	import { api } from '#lib/core/api.ts';
	import { plainMessage as message } from '#lib/core/settings.ts';
	import { toasts } from '#lib/core/toasts.svelte.ts';
	import { linkProblem, mediaKindWord as kindWord, type LinkProblem, type MediaLink, type MediaServer } from './media-link.ts';
	import LinkLogo from './LinkLogo.svelte';
	import { playerStatus } from './player-status.svelte.ts';

	let { servers, links = $bindable(), open = $bindable() }: { servers: MediaServer[]; links: MediaLink[]; open: MediaServer | null } = $props();
	// A link whose server is gone or not answering shows greyed with why in red; one whose sign-in for playing went
	// with a removed server offers Reconnect (linkProblem).
	const why = (p: LinkProblem, kind: string) =>
		p === 'gone' ? t('accounts.watch.problemGone', { server: kindWord(kind) }) : p === 'offline' ? t('accounts.watch.problemOffline', { server: kindWord(kind) }) : p === 'reconnect' ? t('accounts.watch.problemReconnect') : '';

	// Reconnect: the sign-in form for that server, my name there filled in and the tickbox on.
	function reconnect(l: MediaLink) {
		const s = servers.find((x) => x.kind === l.provider && !x.offline) ?? servers.find((x) => x.kind === l.provider);
		if (!s) return;
		shownFor = s.id;
		error = '';
		keep = true;
		name = l.name;
		open = s;
	}
	let name = $state('');
	let typed = $state('');
	let keep = $state(false);
	let busy = $state(false);
	let error = $state('');
	let unlinking = $state('');
	let shownFor: number | null = null;

	// A newly picked server starts with the tickbox as that link already is.
	$effect(() => {
		const id = open?.id ?? null;
		if (id !== shownFor) {
			shownFor = id;
			error = '';
			keep = !!(open && links.find((l) => l.provider === open!.kind)?.kept);
		}
	});

	async function link(e: SubmitEvent) {
		e.preventDefault();
		if (!open) return;
		error = '';
		busy = true;
		try {
			const body: Record<string, unknown> = { name: name.trim(), password: typed };
			if (open.kind === 'jellyfin') body.keep_for_playback = keep;
			const l = await api.post<MediaLink>(`/auth/media-link/${open.id}`, body);
			links = [...links.filter((x) => x.provider !== l.provider), l];
			if (l.provider === 'jellyfin') playerStatus.lost = false;
			toasts.push(t('accounts.watch.linked', { name: l.name, server: open.name }), 'ok');
			open = null;
			name = '';
		} catch (err) {
			error = message(err);
		} finally {
			typed = '';
			busy = false;
		}
	}

	async function unlink(kind: string) {
		busy = true;
		try {
			await api.del(`/auth/media-link/${kind}`);
			links = links.filter((l) => l.provider !== kind);
			if (kind === 'jellyfin') playerStatus.lost = false;
			unlinking = '';
			toasts.push(t('accounts.watch.unlinked'), 'ok');
		} catch (err) {
			toasts.push(message(err), 'fail');
		} finally {
			busy = false;
		}
	}
</script>

{#if links.length}
	<ul class="below my-links" data-testid="media-link">
		{#each links as l (l.provider)}
			{@const problem = linkProblem(l, servers, playerStatus.lost)}
			<li class:gone={problem === 'gone' || problem === 'offline'}>
				<span class="what">
					<span class="small"><LinkLogo kind={l.provider} />{t('accounts.watch.as', { server: kindWord(l.provider), name: l.name })}{#if problem === 'gone'}{` · ${t('accounts.watch.serverGone')}`}{:else if problem === 'offline'}{` · ${t('accounts.watch.serverOffline')}`}{:else if l.kept}{` · ${t('accounts.watch.keptForPlayback')}`}{/if}</span>
					{#if problem}<span class="why" data-testid="media-link-why">{why(problem, l.provider)}</span>{/if}
				</span>
				{#if problem === 'reconnect' && unlinking !== l.provider}
					<Button variant="secondary" type="button" onclick={() => reconnect(l)} data-testid="media-link-reconnect">{t('accounts.watch.reconnect')}</Button>
				{/if}
				{#if unlinking === l.provider}
					<Confirm action="link" onCancel={() => (unlinking = '')}>
						<div class="row acts">
							<Button variant="destructive" type="button" onclick={() => unlink(l.provider)} busy={busy}>{t('accounts.security.unlink')}</Button>
							<Button variant="plain" type="button" onclick={() => (unlinking = '')}>{t('accounts.signin.cancel')}</Button>
						</div>
					</Confirm>
				{:else}
					<Button variant="plain" type="button" onclick={() => (unlinking = l.provider)}>{t('accounts.security.unlink')}</Button>
				{/if}
			</li>
		{/each}
	</ul>
{/if}
{#if open}
	<div class="below">
		<Confirm action="link" onCancel={() => (open = null)}>
			<form class="form" onsubmit={link}>
				<label class="field"><span class="small">{t('accounts.watch.nameOn', { server: open.name })}</span>
					<input class="input" bind:value={name} autocomplete="off" maxlength="128" required /></label>
				<label class="field"><span class="small">{t('accounts.signin.passwordLabel')}</span>
					<input class="input" type="password" bind:value={typed} autocomplete="off" /></label>
				<p class="xs muted">{t('accounts.watch.passwordOnce', { server: open.name })}</p>
				{#if open.kind === 'jellyfin'}
					<label class="keep"><input type="checkbox" bind:checked={keep} data-testid="media-link-keep" /><span class="small">{t('accounts.watch.keepForPlayback')}</span></label>
					<p class="xs muted">{t('accounts.watch.keepForPlaybackHint', { server: open.name })}</p>
				{/if}
				<div class="row acts">
					<Button variant="primary" type="submit" disabled={busy || !name.trim()} busy={busy}>{t('accounts.security.link')}</Button>
					<Button variant="plain" type="button" onclick={() => (open = null)}>{t('accounts.signin.cancel')}</Button>
				</div>
			</form>
		</Confirm>
		{#if error}<p class="form-error" role="alert">{error}</p>{/if}
	</div>
{/if}

<style>
	.my-links { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
	.my-links li { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px; min-width: 0; }
	.form { display: flex; flex-direction: column; gap: 8px; width: min(100%, 360px); }
	.field { display: flex; flex-direction: column; gap: 4px; }
	.keep { display: flex; align-items: center; gap: 8px; }
	.gone .small { opacity: var(--disabled-opacity); }
	.what { display: flex; flex-direction: column; gap: 3px; min-width: 0; margin-inline-end: auto; }
	/* What is wrong, in red (as Library folders say why a folder is not working). */
	.why { font: 500 12px/1.3 var(--font-ui); color: var(--fail-text, var(--fail)); }
	.acts { gap: 8px; flex-wrap: wrap; align-items: center; }
</style>
