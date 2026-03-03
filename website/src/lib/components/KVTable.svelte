<script lang="ts">
	import type { KVPair } from '$lib/api';

	interface Props {
		pairs: KVPair[];
		offset: number;
		emptyMessage?: string;
	}

	let { pairs, offset, emptyMessage = 'No data found.' }: Props = $props();
</script>

<div
	class="overflow-hidden overflow-x-auto rounded-xl border"
	style="border-color: var(--border); background: var(--bg-surface);"
>
	<table class="w-full min-w-[480px] border-collapse text-[0.84rem]">
		<thead>
			<tr class="border-b" style="border-color: var(--border); background: var(--bg-surface2);">
				<th
					class="w-[50px] px-3.5 py-[11px] text-left text-[0.72rem] font-semibold tracking-widest uppercase"
					style="color: var(--text-muted);">#</th
				>
				<th
					class="w-[40%] px-3.5 py-[11px] text-left text-[0.72rem] font-semibold tracking-widest uppercase"
					style="color: var(--text-muted);">Key</th
				>
				<th
					class="px-3.5 py-[11px] text-left text-[0.72rem] font-semibold tracking-widest uppercase"
					style="color: var(--text-muted);">Value</th
				>
			</tr>
		</thead>
		<tbody>
			{#if pairs.length === 0}
				<tr>
					<td
						colspan="3"
						class="px-8 py-16 text-center text-[0.9rem]"
						style="color: var(--text-faint);"
					>
						{emptyMessage}
					</td>
				</tr>
			{:else}
				{#each pairs as pair, i (pair.key)}
					<tr
						class="border-b transition-colors last:border-b-0"
						style="border-color: var(--border-subtle);"
						onmouseenter={(e) =>
							((e.currentTarget as HTMLElement).style.background = 'var(--bg-hover)')}
						onmouseleave={(e) => ((e.currentTarget as HTMLElement).style.background = '')}
					>
						<td
							class="font-jetbrains px-3.5 py-2.5 text-[0.72rem]"
							style="color: var(--text-row-num);">{offset + i + 1}</td
						>
						<td class="px-3.5 py-2.5 align-middle">
							<span
								class="font-jetbrains rounded-[5px] px-2 py-0.5 text-[0.8rem] break-all"
								style="background: var(--violet-bg); color: var(--violet-text);">{pair.key}</span
							>
						</td>
						<td class="px-3.5 py-2.5 align-middle">
							<span
								class="font-jetbrains text-[0.8rem] break-all"
								style="color: var(--text-primary);">{pair.value}</span
							>
						</td>
					</tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>
