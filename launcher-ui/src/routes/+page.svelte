<script lang="ts">
	import { onMount } from 'svelte';

	const DEFAULT_RPC_URL = 'http://localhost:8123/rpc';

	let statusLabel = 'Core: Checking...';
	let statusTone: 'ok' | 'warn' | 'error' = 'warn';

	const rpcUrl = import.meta.env.VITE_GENTRO_RPC_URL as string | undefined;
	const rpcSocket = import.meta.env.VITE_GENTRO_RPC_SOCKET as string | undefined;

	const socketWarning =
		'Core: Socket mode not implemented (set VITE_GENTRO_RPC_URL for HTTP)';

	const resolveRpcTarget = () => {
		if (rpcUrl) {
			return { mode: 'http', target: rpcUrl } as const;
		}
		if (rpcSocket) {
			return { mode: 'socket', target: rpcSocket } as const;
		}
		return { mode: 'http', target: DEFAULT_RPC_URL } as const;
	};

	const fetchStatus = async () => {
		const target = resolveRpcTarget();
		if (target.mode === 'socket') {
			console.warn(socketWarning);
			statusLabel = socketWarning;
			statusTone = 'warn';
			return;
		}

		try {
			const response = await fetch(target.target, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					jsonrpc: '2.0',
					id: 1,
					method: 'core.status',
					params: {}
				})
			});

			if (!response.ok) {
				throw new Error(`HTTP ${response.status}`);
			}

			const payload = await response.json();
			const result = payload?.result;
			if (!result?.name) {
				throw new Error('Invalid response');
			}

			statusLabel = `Core: ${result.status ?? 'ok'} (${result.version ?? 'dev'})`;
			statusTone = result.status === 'ok' ? 'ok' : 'warn';
		} catch (error) {
			console.warn('Core status fetch failed', error);
			statusLabel = 'Core: Unreachable';
			statusTone = 'error';
		}
	};

	onMount(() => {
		void fetchStatus();
	});
</script>

<div class="shell">
	<header class="shell__header">
		<div class="shell__brand">Gentro OS</div>
		<div class="shell__status">System Ready</div>
	</header>
	<main class="shell__main">
		<section class="shell__hero">
			<h1>Launcher Shell</h1>
			<p>Placeholder UI for the console-first gaming experience.</p>
		</section>
	</main>
	<footer class="shell__footer">
		<div class={`shell__badge shell__badge--${statusTone}`}>{statusLabel}</div>
	</footer>
</div>

<style>
	:global(body) {
		margin: 0;
		background: #0d1016;
		color: #e7ecf2;
		font-family: 'Fira Sans', 'Segoe UI', sans-serif;
	}

	:global(*) {
		box-sizing: border-box;
	}

	.shell {
		min-height: 100vh;
		display: grid;
		grid-template-rows: auto 1fr auto;
		background: radial-gradient(circle at top, #1b2330 0%, #0d1016 60%);
	}

	.shell__header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1.5rem 2.5rem;
		border-bottom: 1px solid #1f2a3b;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		font-size: 0.75rem;
	}

	.shell__brand {
		font-weight: 600;
	}

	.shell__status {
		color: #8fb3ff;
	}

	.shell__main {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 3rem 2.5rem;
	}

	.shell__hero {
		max-width: 560px;
		padding: 2.5rem 3rem;
		background: rgba(17, 24, 35, 0.8);
		border: 1px solid #24324a;
		border-radius: 16px;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
	}

	.shell__hero h1 {
		margin: 0 0 0.75rem;
		font-size: 2.4rem;
		letter-spacing: 0.02em;
	}

	.shell__hero p {
		margin: 0;
		color: #b6c1d1;
		line-height: 1.6;
	}

	.shell__footer {
		display: flex;
		justify-content: flex-end;
		padding: 1.25rem 2.5rem 1.75rem;
	}

	.shell__badge {
		padding: 0.4rem 0.9rem;
		border-radius: 999px;
		font-size: 0.75rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		border: 1px solid transparent;
	}

	.shell__badge--ok {
		color: #d2fbe6;
		background: rgba(22, 94, 66, 0.35);
		border-color: rgba(64, 191, 129, 0.6);
	}

	.shell__badge--warn {
		color: #ffe9b0;
		background: rgba(119, 82, 23, 0.35);
		border-color: rgba(237, 176, 65, 0.6);
	}

	.shell__badge--error {
		color: #ffd3d3;
		background: rgba(122, 33, 33, 0.35);
		border-color: rgba(240, 84, 84, 0.6);
	}
</style>
