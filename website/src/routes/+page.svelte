<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import KVExplorer from '$lib/components/KVExplorer.svelte';
	import SeQLEditor from '$lib/components/SeQLEditor.svelte';

	let activeTab: 'explorer' | 'seql' = $state('explorer');
	let refreshTick: number = $state(0);
	let isLight: boolean = $state(false);
</script>

<svelte:head>
	<title>Seele — Key Value Store</title>
	<meta name="description" content="Seele distributed key-value store dashboard" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

<div class="flex h-screen overflow-hidden {isLight ? 'light' : ''}">
	<div class="hidden md:flex">
		<Sidebar
			{activeTab}
			onTabChange={(t) => (activeTab = t)}
			{isLight}
			onThemeToggle={() => (isLight = !isLight)}
		/>
	</div>

	<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
		<header
			class="flex shrink-0 items-center justify-between gap-3 border-b px-4 py-3 md:hidden"
			style="background: var(--bg-surface); border-color: var(--border);"
		>
			<span
				class="bg-gradient-to-br from-[#c4b5fd] to-[#a5b4fc] bg-clip-text text-base font-bold tracking-wide text-transparent"
			>
				Seele
			</span>
			<div class="flex items-center gap-1.5">
				<button
					onclick={() => (activeTab = 'explorer')}
					class="rounded-md border-0 px-3 py-1.5 text-[0.78rem] font-medium transition-all"
					style={activeTab === 'explorer'
						? 'background: rgba(108,99,255,0.15); color: var(--violet-text);'
						: 'background: transparent; color: var(--text-muted);'}>Explorer</button
				>
				<button
					onclick={() => (activeTab = 'seql')}
					class="rounded-md border-0 px-3 py-1.5 text-[0.78rem] font-medium transition-all"
					style={activeTab === 'seql'
						? 'background: rgba(108,99,255,0.15); color: var(--violet-text);'
						: 'background: transparent; color: var(--text-muted);'}>SeQL</button
				>
				<button
					aria-label="Toggle theme"
					onclick={() => (isLight = !isLight)}
					class="ml-1 flex cursor-pointer items-center justify-center rounded-lg border-0 p-1.5"
					style="background: var(--bg-hover); color: var(--text-muted);"
				>
					{#if isLight}
						<svg width="15" height="15" viewBox="0 0 24 24" fill="none"
							><path
								d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							/></svg
						>
					{:else}
						<svg width="15" height="15" viewBox="0 0 24 24" fill="none"
							><circle cx="12" cy="12" r="5" stroke="currentColor" stroke-width="2" /><path
								d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
							/></svg
						>
					{/if}
				</button>
			</div>
		</header>

		<main class="min-w-0 flex-1 overflow-auto p-4 md:p-8" style="background: var(--bg-app);">
			{#if activeTab === 'explorer'}
				<KVExplorer {refreshTick} />
			{:else}
				<SeQLEditor onMutated={() => refreshTick++} />
			{/if}
		</main>
	</div>
</div>
