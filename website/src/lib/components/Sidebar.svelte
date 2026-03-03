<script lang="ts">
	interface Props {
		activeTab: 'explorer' | 'seql';
		onTabChange: (tab: 'explorer' | 'seql') => void;
		isLight: boolean;
		onThemeToggle: () => void;
	}

	let { activeTab, onTabChange, isLight, onThemeToggle }: Props = $props();
</script>

<aside
	class="flex h-screen w-[220px] shrink-0 flex-col overflow-hidden border-r"
	style="background: var(--bg-surface); border-color: var(--border);"
>
	<div
		class="flex items-center gap-2.5 border-b px-[18px] py-5"
		style="border-color: var(--border);"
	>
		<img src="/assets/seele_logo.png" alt="seele_logo" width="16" height="16" />
		<span
			class="bg-gradient-to-br from-[#c4b5fd] to-[#a5b4fc] bg-clip-text text-lg font-bold tracking-wide text-transparent"
		>
			Seele
		</span>
	</div>

	<nav class="flex flex-1 flex-col gap-0.5 p-2.5">
		<button
			id="tab-explorer"
			class="flex w-full cursor-pointer items-center gap-2.5 rounded-lg border-0 px-2.5 py-[9px] text-left text-sm font-medium transition-all"
			style={activeTab === 'explorer'
				? 'background: rgba(108,99,255,0.15); color: var(--violet-text);'
				: 'background: transparent; color: var(--text-muted);'}
			onclick={() => onTabChange('explorer')}
		>
			<svg width="18" height="18" viewBox="0 0 24 24" fill="none">
				<rect x="3" y="3" width="18" height="18" rx="2" stroke="currentColor" stroke-width="2" />
				<path d="M3 9h18M3 15h18M9 3v18" stroke="currentColor" stroke-width="2" />
			</svg>
			<span>KV Explorer</span>
		</button>

		<button
			id="tab-seql"
			class="flex w-full cursor-pointer items-center gap-2.5 rounded-lg border-0 px-2.5 py-[9px] text-left text-sm font-medium transition-all"
			style={activeTab === 'seql'
				? 'background: rgba(108,99,255,0.15); color: var(--violet-text);'
				: 'background: transparent; color: var(--text-muted);'}
			onclick={() => onTabChange('seql')}
		>
			<svg width="18" height="18" viewBox="0 0 24 24" fill="none">
				<polyline
					points="16 18 22 12 16 6"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
				<polyline
					points="8 6 2 12 8 18"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
			<span>SeQL Editor</span>
		</button>
	</nav>

	<div class="border-t p-3" style="border-color: var(--border);">
		<button
			id="theme-toggle"
			onclick={onThemeToggle}
			class="flex w-full cursor-pointer items-center gap-2 rounded-lg border-0 px-2.5 py-2 text-[0.8rem] font-medium transition-all"
			style="background: var(--bg-hover); color: var(--text-muted);"
		>
			{#if isLight}
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none">
					<path
						d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
				</svg>
				Dark mode
			{:else}
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none">
					<circle cx="12" cy="12" r="5" stroke="currentColor" stroke-width="2" />
					<path
						d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
					/>
				</svg>
				Light mode
			{/if}
		</button>
	</div>
</aside>
