<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { toast } from 'svelte-sonner';
	import Icon from '$lib/components/Icon.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Field from '$lib/components/ui/field.svelte';
	import Input from '$lib/components/ui/input.svelte';
	import Select from '$lib/components/ui/select.svelte';
	import { api } from '$lib/backend';
	import { spaces } from '$lib/spaces.svelte';
	import type { Space, SpaceMember, SpaceRole } from '$lib/types';
	import { formatDateTime } from '$lib/utils';

	const spaceId = $derived(page.params.id ?? '');

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	let name = $state('');
	let description = $state('');
	let savingSpace = $state(false);

	let newMember = $state('');
	let newRole = $state<SpaceRole>('member');
	let addingMember = $state(false);

	const roleOptions = [
		{ value: 'owner', label: 'Owner' },
		{ value: 'admin', label: 'Admin' },
		{ value: 'member', label: 'Member' }
	];

	async function load() {
		if (!spaceId) return;
		loading = true;
		try {
			const payload = await api.space(spaceId);
			space = payload.space;
			members = payload.members;
			name = payload.space.name;
			description = payload.space.description;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to load the space');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void spaceId;
		void load();
	});

	async function saveSpace() {
		savingSpace = true;
		try {
			const updated = await spaces.update(spaceId, {
				name: name.trim(),
				description: description.trim()
			});
			space = updated;
			toast.success('Space updated.');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to update the space');
		} finally {
			savingSpace = false;
		}
	}

	async function removeSpace() {
		try {
			await spaces.remove(spaceId);
			toast.success('Space deleted.');
			await goto('/spaces', { replaceState: true });
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to delete the space');
		}
	}

	async function addMember() {
		if (!newMember.trim()) {
			toast.error('A member needs an identifier.');
			return;
		}

		addingMember = true;
		try {
			const payload = await api.addMember(spaceId, newMember.trim(), newRole);
			// The API returns the existing membership unchanged when the person is
			// already in, so replace rather than append.
			members = [
				...members.filter((member) => member.userId !== payload.member.userId),
				payload.member
			];
			newMember = '';
			toast.success('Member added.');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to add the member');
		} finally {
			addingMember = false;
		}
	}

	async function changeRole(userId: string, role: SpaceRole) {
		try {
			const payload = await api.updateMember(spaceId, userId, role);
			members = members.map((member) => (member.userId === userId ? payload.member : member));
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to change the role');
		}
	}

	async function removeMember(userId: string) {
		try {
			await api.removeMember(spaceId, userId);
			members = members.filter((member) => member.userId !== userId);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to remove the member');
		}
	}
</script>

<div class="grid gap-5">
	<a
		href="/spaces"
		class="inline-flex w-fit items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
	>
		<Icon icon="solar:arrow-left-linear" size={15} />
		Back to spaces
	</a>

	{#if loading}
		<p class="text-sm text-muted-foreground">Loading…</p>
	{:else if !space}
		<p class="text-sm text-muted-foreground">This space no longer exists.</p>
	{:else}
		<div class="flex flex-wrap items-center gap-3">
			<span class="flex size-10 items-center justify-center rounded-lg border border-border text-muted-foreground">
				<Icon icon="solar:buildings-2-linear" size={20} />
			</span>
			<h2 class="font-heading text-xl font-semibold tracking-tight">{space.name}</h2>
			<Badge variant="outline">{members.length} {members.length === 1 ? 'member' : 'members'}</Badge>
		</div>

		<section class="grid max-w-2xl gap-4 rounded-xl border border-border bg-card p-5">
			<h3 class="font-heading text-base font-semibold">Details</h3>

			<Field label="Name" required>
				<Input bind:value={name} />
			</Field>

			<Field label="Description">
				<Input bind:value={description} />
			</Field>

			<div class="flex flex-wrap justify-between gap-2">
				<Button onclick={saveSpace} disabled={savingSpace}>
					<Icon icon="solar:diskette-linear" size={15} />
					{savingSpace ? 'Saving…' : 'Save'}
				</Button>
				<Button variant="destructive" onclick={removeSpace}>
					<Icon icon="solar:trash-bin-trash-linear" size={15} />
					Delete space
				</Button>
			</div>
		</section>

		<section class="grid gap-4">
			<h3 class="font-heading text-base font-semibold">Members</h3>

			<div class="flex flex-wrap items-end gap-2">
				<Field label="Identifier" class="min-w-56 flex-1">
					<Input bind:value={newMember} placeholder="someone@facile.studio" />
				</Field>
				<Field label="Role" class="w-40">
					<Select bind:value={newRole as string} options={roleOptions} />
				</Field>
				<Button onclick={addMember} disabled={addingMember}>
					<Icon icon="solar:add-circle-linear" size={15} />
					Add
				</Button>
			</div>

			{#if members.length === 0}
				<p class="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted-foreground">
					No member yet.
				</p>
			{:else}
				<div class="grid gap-2">
					{#each members as member (member.userId)}
						<div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card px-4 py-3">
							<div class="flex min-w-0 items-center gap-2.5">
								<Icon icon="solar:shield-user-linear" size={18} class="text-muted-foreground" />
								<div class="grid min-w-0">
									<strong class="truncate text-sm font-medium">{member.userId}</strong>
									<span class="text-xs text-muted-foreground">
										Joined {formatDateTime(member.joinedAt)}
									</span>
								</div>
							</div>

							<div class="flex items-center gap-2">
								<Select
									value={member.role}
									options={roleOptions}
									class="w-36"
									onchange={(event) =>
										changeRole(member.userId, (event.currentTarget as HTMLSelectElement).value as SpaceRole)}
								/>
								<Button
									variant="destructive"
									size="icon"
									onclick={() => removeMember(member.userId)}
									title="Remove"
								>
									<Icon icon="solar:trash-bin-trash-linear" size={14} />
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</section>
	{/if}
</div>
