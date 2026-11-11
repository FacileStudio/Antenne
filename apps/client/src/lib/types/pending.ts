/** One replay an AI client asked for, waiting on the confirm page. */
export interface PendingAction {
	id: string;
	kind: string;
	client_id: string;
	summary: string;
	created_at: string;
	expires_at: string;
	state: 'ready' | 'changed' | 'missing';
	event?: {
		id: string;
		at: string;
		source: string;
		provider: string;
		event: string;
		title?: string;
	};
	target?: {
		id: string;
		name: string;
		type: string;
		label: string;
	};
}
