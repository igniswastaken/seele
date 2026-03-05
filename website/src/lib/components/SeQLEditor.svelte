<script lang="ts">
	import { runQuery } from '$lib/api';

	const MUTATING = ['PLANT', 'BANISH', 'MORPH'];

	interface Props {
		onMutated?: () => void;
	}

	let { onMutated }: Props = $props();

	const REVEAL_INITIAL = 20;

	let query: string = $state('REVEAL *');
	let singleResult: Record<string, unknown> | null = $state(null);
	let result: string = $state('');
	let streamRows: any[] = $state([]);
	let streamShown: number = $state(0);
	let error: string = $state('');
	let loading: boolean = $state(false);
	let history: string[] = $state([]);

	let hasMore = $derived(streamShown < streamRows.length);

	function revealNext() {
		if (hasMore) streamShown = Math.min(streamShown + 1, streamRows.length);
	}

	async function execute() {
		if (!query.trim()) return;
		loading = true;
		error = '';
		result = '';
		singleResult = null;
		streamRows = [];
		streamShown = 0;
		try {
			const res = await runQuery(query);
			if (res.error) {
				error = res.error;
			} else {
				history = [query, ...history.filter((h) => h !== query)].slice(0, 10);
				if (MUTATING.some((cmd) => query.trim().toUpperCase().startsWith(cmd))) {
					onMutated?.();
				}
				if (Array.isArray(res.result)) {
					const dedupMap = new Map<string, any>();
					const nonKeyedRows = [];
					let hasKeys = false;

					for (const row of res.result) {
						if (row && typeof row === 'object' && 'key' in row) {
							dedupMap.set(row.key, row);
							hasKeys = true;
						} else {
							nonKeyedRows.push(row);
						}
					}

					streamRows = hasKeys ? Array.from(dedupMap.values()).concat(nonKeyedRows) : res.result;
					streamShown = Math.min(REVEAL_INITIAL, streamRows.length);
				} else if (res.result !== null && typeof res.result === 'object') {
					singleResult = res.result as Record<string, unknown>;
				} else {
					result = String(res.result);
				}
			}
		} catch (e: any) {
			error = e.message ?? 'Network error';
		} finally {
			loading = false;
		}
	}

	function onKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
			e.preventDefault();
			execute();
			return;
		}
		if (e.key === 'Enter' && e.target !== document.getElementById('seql-editor') && hasMore) {
			e.preventDefault();
			revealNext();
		}
	}
</script>

<div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
	<div>
		<h1 class="text-2xl font-bold tracking-tight" style="color: var(--text-primary);">
			SeQL Editor
		</h1>
		<p class="mt-0.5 text-[0.82rem]" style="color: var(--text-muted);">
			Run queries against the Seele key-value store using Seele Query Language (SeQL)
		</p>
	</div>
	<button
		id="run-query-btn"
		disabled={loading}
		onclick={execute}
		class="flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg border-0 px-4 py-2 text-[0.82rem] font-semibold whitespace-nowrap text-white transition-all hover:-translate-y-px disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto"
		style="background: linear-gradient(135deg, var(--accent-from), var(--accent-to));"
	>
		{#if loading}
			<div
				class="h-[13px] w-[13px] animate-spin rounded-full border-2 border-white/30 border-t-white"
			></div>
			Running…
		{:else}
			<svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor">
				<polygon points="5 3 19 12 5 21 5 3" />
			</svg>
			Run
			<kbd
				class="font-jetbrains rounded px-[5px] py-px text-[0.68rem]"
				style="background: rgba(255,255,255,0.15);">Ctrl+Enter</kbd
			>
		{/if}
	</button>
</div>

<div class="mb-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
	<div
		class="flex min-h-[220px] flex-col overflow-hidden rounded-xl border sm:min-h-[340px]"
		style="border-color: var(--border); background: var(--bg-surface);"
	>
		<div
			class="border-b px-3.5 py-2.5 text-[0.72rem] font-semibold tracking-widest uppercase"
			style="border-color: var(--border); background: var(--bg-surface2); color: var(--text-muted);"
		>
			Query
		</div>
		<textarea
			id="seql-editor"
			bind:value={query}
			onkeydown={onKeydown}
			spellcheck={false}
			placeholder={'REVEAL key1\nPLANT key1 WITH "value1"\nBANISH key1\nMORPH key1 TO "new_value"'}
			class="font-jetbrains min-h-[200px] flex-1 resize-none border-0 bg-transparent p-4 text-[0.88rem] leading-relaxed outline-none"
			style="color: var(--violet-text); caret-color: var(--accent-from);"
		></textarea>

		{#if history.length > 0}
			<div class="border-t px-3.5 py-2.5" style="border-color: var(--border);">
				<div
					class="mb-1.5 text-[0.68rem] font-semibold tracking-widest uppercase"
					style="color: var(--text-faint);"
				>
					Recent
				</div>
				<div class="flex flex-wrap gap-1.5">
					{#each history as h}
						<button
							onclick={() => (query = h)}
							title={h}
							class="font-jetbrains max-w-[180px] cursor-pointer overflow-hidden rounded-[5px] border px-2 py-0.5 text-[0.73rem] text-ellipsis whitespace-nowrap transition-all"
							style="border-color: var(--border); background: var(--bg-surface2); color: var(--text-muted);"
						>
							{h}
						</button>
					{/each}
				</div>
			</div>
		{/if}
	</div>

	<div
		class="flex min-h-[340px] flex-col overflow-hidden rounded-xl border"
		style="border-color: var(--border); background: var(--bg-surface);"
	>
		<div
			class="border-b px-3.5 py-2.5 text-[0.72rem] font-semibold tracking-widest uppercase"
			style="border-color: var(--border); background: var(--bg-surface2); color: var(--text-muted);"
		>
			Result
		</div>
		{#if error}
			<div
				class="m-3.5 flex items-center gap-2 rounded-lg border px-3.5 py-2.5 text-[0.82rem]"
				style="border-color: var(--red-border); background: var(--red-bg); color: var(--red-text);"
			>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none">
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
		{:else if streamRows.length > 0}
			<div class="flex-1 overflow-auto">
				<table class="w-full border-collapse text-[0.8rem]">
					<thead class="sticky top-0">
						<tr
							class="border-b"
							style="border-color: var(--border); background: var(--bg-surface2);"
						>
							<th
								class="w-8 px-3 py-2 text-left text-[0.7rem] font-semibold tracking-widest uppercase"
								style="color: var(--text-muted);">#</th
							>
							<th
								class="w-[40%] px-3 py-2 text-left text-[0.7rem] font-semibold tracking-widest uppercase"
								style="color: var(--text-muted);">Key</th
							>
							<th
								class="px-3 py-2 text-left text-[0.7rem] font-semibold tracking-widest uppercase"
								style="color: var(--text-muted);">Value</th
							>
						</tr>
					</thead>
					<tbody>
						{#each streamRows.slice(0, streamShown) as row, i}
							<tr
								class="border-b transition-colors last:border-b-0"
								style="border-color: var(--border-subtle);"
								onmouseenter={(e) =>
									((e.currentTarget as HTMLElement).style.background = 'var(--bg-hover)')}
								onmouseleave={(e) => ((e.currentTarget as HTMLElement).style.background = '')}
							>
								<td
									class="font-jetbrains px-3 py-2 text-[0.7rem]"
									style="color: var(--text-row-num);">{i + 1}</td
								>
								<td class="px-3 py-2 align-middle">
									<span
										class="font-jetbrains rounded-[4px] px-1.5 py-0.5 text-[0.78rem]"
										style="background: var(--violet-bg); color: var(--violet-text);"
										>{row.key ?? '—'}</span
									>
								</td>
								<td
									class="font-jetbrains px-3 py-2 align-middle text-[0.78rem] break-all"
									style="color: var(--green-text);"
									>{row.found === false ? '(not found)' : (row.value ?? '—')}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			{#if hasMore}
				<button
					onclick={revealNext}
					class="flex w-full cursor-pointer items-center justify-center gap-2 border-t px-4 py-2.5 text-[0.78rem] transition-all"
					style="border-color: var(--border); background: var(--bg-surface); color: var(--text-muted);"
				>
					<svg width="13" height="13" viewBox="0 0 24 24" fill="none">
						<polyline
							points="6 9 12 15 18 9"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
					Reveal next
					<span
						class="rounded px-1.5 py-0.5 text-[0.68rem]"
						style="background: var(--bg-surface2); color: var(--text-faint);"
					>
						{streamShown} / {streamRows.length}
					</span>
					<kbd
						class="font-jetbrains rounded px-1.5 py-0.5 text-[0.65rem]"
						style="background: var(--bg-surface2); color: var(--text-faint);">Enter</kbd
					>
				</button>
			{:else}
				<div
					class="flex items-center justify-center gap-1.5 border-t py-2 text-[0.75rem]"
					style="border-color: var(--border); color: var(--green-text);"
				>
					<svg width="12" height="12" viewBox="0 0 24 24" fill="none">
						<polyline
							points="20 6 9 17 4 12"
							stroke="currentColor"
							stroke-width="2.5"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
					All {streamRows.length} results revealed
				</div>
			{/if}
		{:else if singleResult}
			<div class="flex flex-1 flex-col gap-2 p-4">
				{#if singleResult.found === false}
					<div
						class="flex items-center gap-2 rounded-lg border px-3.5 py-2.5 text-[0.82rem]"
						style="border-color: var(--amber-border); background: var(--amber-bg); color: var(--amber-text);"
					>
						<svg width="13" height="13" viewBox="0 0 24 24" fill="none"
							><circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" /><line
								x1="12"
								y1="8"
								x2="12"
								y2="12"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
							/><line
								x1="12"
								y1="16"
								x2="12.01"
								y2="16"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
							/></svg
						>
						Key
						<span
							class="font-jetbrains mx-1 rounded px-1.5 py-0.5"
							style="background: var(--amber-bg); color: var(--amber-text);"
							>{singleResult.key}</span
						> not found
					</div>
				{:else}
					<div class="flex flex-col gap-1">
						<span
							class="text-[0.68rem] font-semibold tracking-widest uppercase"
							style="color: var(--text-muted);">Key</span
						>
						<span
							class="font-jetbrains rounded-[4px] px-2 py-1 text-[0.85rem]"
							style="background: var(--violet-bg); color: var(--violet-text);"
							>{singleResult.key}</span
						>
					</div>
					<div class="flex flex-col gap-1">
						<span
							class="text-[0.68rem] font-semibold tracking-widest uppercase"
							style="color: var(--text-muted);">Value</span
						>
						<span
							class="font-jetbrains rounded-[4px] px-2 py-1 text-[0.85rem] break-all"
							style="background: var(--bg-surface2); color: var(--green-text);"
							>{singleResult.value}</span
						>
					</div>
				{/if}
			</div>
		{:else if result}
			<pre
				class="font-jetbrains flex-1 overflow-auto p-4 text-[0.82rem] leading-relaxed break-all whitespace-pre-wrap"
				style="color: var(--green-text);">{result}</pre>
		{:else}
			<div
				class="flex flex-1 flex-col items-center justify-center gap-2.5 text-[0.82rem]"
				style="color: var(--text-faint);"
			>
				<svg width="40" height="40" viewBox="0 0 24 24" fill="none" class="opacity-30">
					<polyline
						points="16 18 22 12 16 6"
						stroke="currentColor"
						stroke-width="1.5"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
					<polyline
						points="8 6 2 12 8 18"
						stroke="currentColor"
						stroke-width="1.5"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
				</svg>
				<p>Run a query to see results</p>
			</div>
		{/if}
	</div>
</div>

<div
	class="rounded-xl border px-5 py-4"
	style="border-color: var(--border); background: var(--bg-surface);"
>
	<div
		class="mb-3 text-[0.72rem] font-semibold tracking-widest uppercase"
		style="color: var(--text-muted);"
	>
		SeQL REFERENCE
	</div>
	<div class="grid gap-2.5" style="grid-template-columns: repeat(auto-fill, minmax(280px, 1fr))">
		{#each [{ cmd: 'REVEAL', lines: ['REVEAL key', 'REVEAL (key1, key2)', 'REVEAL PREFIX user:', 'REVEAL *'] }, { cmd: 'PLANT', lines: ['PLANT key WITH value', 'PLANT (key1 WITH val1, key2 WITH val2)'] }, { cmd: 'BANISH', lines: ['BANISH key', 'BANISH (key1, key2)'] }, { cmd: 'MORPH', lines: ['MORPH key TO newvalue', 'MORPH (key1 TO val1, key2 TO val2)'] }] as row}
			<div class="flex flex-col gap-1">
				<span class="font-jetbrains text-[0.76rem] font-bold" style="color: var(--violet-text);"
					>{row.cmd}</span
				>
				{#each row.lines as line}
					<code class="font-jetbrains text-[0.73rem]" style="color: var(--text-muted);">{line}</code
					>
				{/each}
			</div>
		{/each}
	</div>
	<p class="mt-3 text-[0.68rem]" style="color: var(--text-faint);">
		Semicolons are optional. Quote keys with special characters: <code
			class="font-jetbrains"
			style="color: var(--text-muted);">'my:key'</code
		>
	</p>
</div>
