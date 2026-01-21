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
		grid-template-rows: auto 1fr;
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
</style>
