<script lang="ts">
  import { launchStore } from '$lib/stores/launch.svelte.js';
  import { gamesStore } from '$lib/stores/games.svelte.js';
  import { LaunchStatus } from '../../bindings/github.com/rhythmerc/gentro-ui/services/games/models/models.js';
</script>

<div class="app-container">
  <header class="page-header">
    <h1 class="page-title">Game Library</h1>
    <p class="game-count">
      {#if !gamesStore.loading}
        {gamesStore.games.length} games
      {:else}
        Loading...
      {/if}
    </p>
  </header>

  <main class="main-content">
    <slot />
  </main>

  <footer class="launch-footer">
    <div class="active-games">
      {#if launchStore.hasActiveLaunches}
        {#if launchStore.activeLaunches.length === 1}
          {@const launch = launchStore.activeLaunches[0]}
          <div class="launch-item">
            <span class="game-name">{launch.gameName}</span>
            <span class="launch-status" class:running={launch.status === LaunchStatus.LaunchStatusRunning}>
              {#if launch.status === LaunchStatus.LaunchStatusLaunching}
                Launching...
              {:else}
                Running
              {/if}
            </span>
          </div>
        {:else}
          {@const firstLaunch = launchStore.activeLaunches[0]}
          {@const extraCount = launchStore.activeLaunches.length - 1}
          <div class="launch-item">
            <span class="game-name">{firstLaunch.gameName}</span>
            <span class="extra-count">+{extraCount} more</span>
            <span class="launch-status" class:running={firstLaunch.status === LaunchStatus.LaunchStatusRunning}>
              {#if firstLaunch.status === LaunchStatus.LaunchStatusLaunching}
                Launching...
              {:else}
                Running
              {/if}
            </span>
          </div>
        {/if}
      {/if}
    </div>
  </footer>
</div>

<style>
  .app-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }

  .page-header {
    flex-shrink: 0;
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    padding: 1.5rem 2rem;
    background: #0f172a;
    border-bottom: 1px solid rgba(148, 163, 184, 0.2);
  }

  .page-title {
    margin: 0;
    font-size: 1.75rem;
    font-weight: 700;
    background: linear-gradient(90deg, #60a5fa 0%, #a78bfa 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }

  .game-count {
    margin: 0;
    color: #94a3b8;
    font-size: 1rem;
  }

  .main-content {
    flex: 1;
    overflow-y: auto;
    padding: 2rem;
  }

  .launch-footer {
    flex-shrink: 0;
    padding: 12px 24px;
    background: #1e293b;
    border-top: 1px solid rgba(148, 163, 184, 0.2);
  }

  .active-games {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
    align-items: center;
    min-height: 32px;
  }

  .launch-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 6px 12px;
    background: rgba(30, 41, 59, 0.8);
    border-radius: 6px;
    border: 1px solid rgba(148, 163, 184, 0.2);
  }

  .game-name {
    font-weight: 600;
    color: #f1f5f9;
    font-size: 0.9rem;
  }

  .launch-status {
    color: #60a5fa;
    font-size: 0.8rem;
    font-weight: 500;
  }

  .launch-status.running {
    color: #4ade80;
  }

  .extra-count {
    color: #94a3b8;
    font-size: 0.8rem;
    font-weight: 500;
    margin-left: -0.25rem;
  }
</style>
