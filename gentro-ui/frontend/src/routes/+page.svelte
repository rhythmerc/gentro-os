<script lang="ts">
  import { onMount } from 'svelte';
  import GameCard from '../lib/components/GameCard.svelte';
  import { gamesStore } from '../lib/stores/games.svelte.js';
  import { launchStore } from '../lib/stores/launch.svelte.js';
  
  const store = gamesStore;
  
  onMount(() => {
    store.loadGames();
  });

  function handleGameClick(instanceId: string) {
    launchStore.launchGame(instanceId);
  }
</script>

<div class="games-container">
  {#if store.loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading your games...</p>
    </div>
  {:else if store.error}
    <div class="error-state">
      <p class="error-message">{store.error}</p>
      <button class="retry-button" onclick={() => store.loadGames()}>
        Retry
      </button>
    </div>
  {:else if store.games.length === 0}
    <div class="empty-state">
      <p class="empty-title">No games found</p>
      <p class="empty-subtitle">Add some games to get started</p>
    </div>
  {:else}
    <div class="games-grid">
      {#each store.games as gameWithInstance}
        <GameCard 
          {gameWithInstance}
          artUrl={store.getArtUrl(gameWithInstance.instance.id, 'header')}
          formatFileSize={store.formatFileSize}
          onClick={() => handleGameClick(gameWithInstance.instance.id)}
        />
      {/each}
    </div>
  {/if}
</div>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background: #0f172a;
    color: #f1f5f9;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
  }
  
  .games-container {
    width: 100%;
  }
  
  .games-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    grid-auto-rows: auto;
    gap: 1.5rem;
  }
  
  @media (max-width: 640px) {
    .games-grid {
      grid-template-columns: 1fr;
    }
  }
  
  @media (min-width: 641px) and (max-width: 1024px) {
    .games-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
  
  @media (min-width: 1025px) and (max-width: 1440px) {
    .games-grid {
      grid-template-columns: repeat(3, 1fr);
    }
  }
  
  @media (min-width: 1441px) {
    .games-grid {
      grid-template-columns: repeat(4, 1fr);
    }
  }
  
  .loading-state,
  .error-state,
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 400px;
    text-align: center;
  }
  
  .spinner {
    width: 48px;
    height: 48px;
    border: 3px solid rgba(148, 163, 184, 0.2);
    border-top-color: #60a5fa;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 1rem;
  }
  
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  
  .error-message {
    color: #f87171;
    margin-bottom: 1rem;
  }
  
  .retry-button {
    padding: 0.75rem 1.5rem;
    background: #3b82f6;
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    cursor: pointer;
    transition: background 0.2s;
  }
  
  .retry-button:hover {
    background: #2563eb;
  }
  
  .empty-title {
    font-size: 1.5rem;
    font-weight: 600;
    color: #f1f5f9;
    margin: 0 0 0.5rem 0;
  }
  
  .empty-subtitle {
    color: #64748b;
    margin: 0;
  }
</style>
