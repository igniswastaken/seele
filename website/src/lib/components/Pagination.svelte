<script lang="ts">
	interface Props {
		currentPage: number;
		totalPages: number;
		onPageChange: (page: number) => void;
	}

	let { currentPage, totalPages, onPageChange }: Props = $props();

	function pages(): (number | '...')[] {
		const result: (number | '...')[] = [];
		const delta = 2;
		const left = currentPage - delta;
		const right = currentPage + delta;
		for (let i = 1; i <= totalPages; i++) {
			if (i === 1 || i === totalPages || (i >= left && i <= right)) {
				result.push(i);
			} else if (i === left - 1 || i === right + 1) {
				result.push('...');
			}
		}
		return result;
	}
</script>

{#if totalPages > 1}
	<div class="mt-5 flex items-center justify-center gap-1.5">
		<button
			id="page-prev"
			class="flex h-[34px] min-w-[34px] cursor-pointer items-center justify-center rounded-[7px] border px-2.5 text-[0.82rem] transition-all disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-30"
			style="border-color: var(--border); background: var(--bg-surface); color: var(--text-muted);"
			onclick={() => onPageChange(currentPage - 1)}
			disabled={currentPage === 1}
		>
			← Prev
		</button>

		<div class="flex items-center gap-1">
			{#each pages() as page}
				{#if page === '...'}
					<span class="px-1 text-[0.82rem]" style="color: var(--text-faint);">…</span>
				{:else}
					<button
						id="page-{page}"
						class="flex h-[34px] min-w-[34px] cursor-pointer items-center justify-center rounded-[7px] border px-2.5 text-[0.82rem] transition-all"
						style={currentPage === page
							? 'border: none; background: linear-gradient(135deg, var(--accent-from), var(--accent-to)); color: white; font-weight: 600;'
							: 'border-color: var(--border); background: var(--bg-surface); color: var(--text-muted);'}
						onclick={() => onPageChange(page as number)}
					>
						{page}
					</button>
				{/if}
			{/each}
		</div>

		<button
			id="page-next"
			class="flex h-[34px] min-w-[34px] cursor-pointer items-center justify-center rounded-[7px] border px-2.5 text-[0.82rem] transition-all disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-30"
			style="border-color: var(--border); background: var(--bg-surface); color: var(--text-muted);"
			onclick={() => onPageChange(currentPage + 1)}
			disabled={currentPage === totalPages}
		>
			Next →
		</button>
	</div>
{/if}
