<script lang="ts">
	import { fetchPage, PAGE_SIZE, type KVPair } from '$lib/api';
	import KVTable from './KVTable.svelte';
	import Pagination from './Pagination.svelte';

	interface Props {
		refreshTick?: number;
	}
	let { refreshTick = 0 }: Props = $props();

	let pairs: KVPair[] = $state([]);
	let total: number = $state(0);
	let currentPage: number = $state(1);
	let loading: boolean = $state(false);
	let error: string = $state('');
	let searchQuery: string = $state('');

	let totalPages = $derived(Math.max(1, Math.ceil(total / PAGE_SIZE)));

	let uniquePairs = $derived(
		Array.from(pairs.reduce((map, p) => map.set(p.key, p), new Map<string, KVPair>()).values())
	);

	let visiblePairs = $derived(
		searchQuery.trim()
			? uniquePairs.filter(
					(p) =>
						p.key.toLowerCase().includes(searchQuery.toLowerCase()) ||
						p.value.toLowerCase().includes(searchQuery.toLowerCase())
				)
			: uniquePairs
	);

	async function loadPage(page: number) {
		loading = true;
		error = '';
		try {
			const result = await fetchPage(page);
			pairs = result.pairs;
			total = result.total;
			currentPage = page;
		} catch (e: any) {
			error = e.message ?? 'Unknown error';
		} finally {
			loading = false;
		}
	}

	function onPageChange(page: number) {
		loadPage(page);
	}

	$effect(() => {
		refreshTick;
		loadPage(currentPage);
	});
</script>

<div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
	<div>
		<h1 class="text-2xl font-bold tracking-tight" style="color: var(--text-primary);">
			KV Explorer
		</h1>
		<p class="mt-0.5 text-[0.82rem]" style="color: var(--text-muted);">
			{#if loading}Loading…
			{:else}{total} total entries · page {currentPage} of {totalPages}{/if}
		</p>
	</div>
	<div class="flex shrink-0 items-center gap-2.5">
		<div class="relative flex-1 sm:flex-none">
			<svg
				class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2"
				style="color: var(--text-muted);"
				width="16"
				height="16"
				viewBox="0 0 24 24"
				fill="none"
			>
				<circle cx="11" cy="11" r="8" stroke="currentColor" stroke-width="2" />
				<path d="m21 21-4.35-4.35" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
			</svg>
			<input
				id="search-input"
				type="text"
				placeholder="Filter this page…"
				bind:value={searchQuery}
				class="w-full rounded-lg border py-2 pr-3 pl-[34px] text-[0.82rem] transition-colors focus:ring-2 focus:outline-none sm:w-52"
				style="border-color: var(--border); background: var(--bg-surface); color: var(--text-primary);"
			/>
		</div>
		<button
			id="refresh-btn"
			disabled={loading}
			onclick={() => loadPage(currentPage)}
			class="flex shrink-0 cursor-pointer items-center gap-1.5 rounded-lg border-0 px-4 py-2 text-[0.82rem] font-semibold text-white transition-all hover:-translate-y-px disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50"
			style="background: linear-gradient(135deg, var(--accent-from), var(--accent-to));"
		>
			<svg
				width="15"
				height="15"
				viewBox="0 0 24 24"
				fill="none"
				class={loading ? 'animate-spin' : ''}
			>
				<polyline
					points="23 4 23 10 17 10"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
				<polyline
					points="1 20 1 14 7 14"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
				<path
					d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
			Refresh
		</button>
	</div>
</div>

{#if error}
	<div
		class="mb-4 flex items-center gap-2 rounded-lg border px-3.5 py-2.5 text-[0.82rem]"
		style="border-color: var(--red-border); background: var(--red-bg); color: var(--red-text);"
	>
		<svg width="16" height="16" viewBox="0 0 24 24" fill="none">
			<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" />
			<line
				x1="12"
				y1="8"
				x2="12"
				y2="12"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
			/>
			<line
				x1="12"
				y1="16"
				x2="12.01"
				y2="16"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
			/>
		</svg>
		{error}
	</div>
{/if}

{#if loading}
	<div
		class="flex items-center justify-center gap-3 py-20 text-[0.9rem]"
		style="color: var(--text-muted);"
	>
		<div
			class="h-[22px] w-[22px] animate-spin rounded-full border-2"
			style="border-color: var(--border); border-top-color: var(--accent-from);"
		></div>
		<span>Fetching data…</span>
	</div>
{:else}
	<KVTable
		pairs={visiblePairs}
		offset={(currentPage - 1) * PAGE_SIZE}
		emptyMessage={searchQuery
			? 'No results match your filter.'
			: 'No data found. Start by adding some keys.'}
	/>
	<Pagination {currentPage} {totalPages} {onPageChange} />
{/if}
