<script lang="ts">
  import type { GameWithInstance } from '../../../bindings/github.com/rhythmerc/gentro-ui/services/games/models/models.js';
  import { launchStore } from '../stores/launch.svelte.js';
  
  let { gameWithInstance, artUrl, formatFileSize, onClick }: {
    gameWithInstance: GameWithInstance;
    artUrl: string;
    formatFileSize: (bytes: number) => string;
    onClick?: () => void;
  } = $props();
  
  const game = $derived(gameWithInstance.game);
  const instance = $derived(gameWithInstance.instance);
  
  const isLaunching = $derived(launchStore.isLaunching(instance.id));
  
  function handleClick() {
    if (onClick) {
      onClick();
    }
  }
  
  function handleImageError(event: Event) {
    const img = event.target as HTMLImageElement;
    img.style.display = 'none';
  }
  
  function handleImageLoad(event: Event) {
    const img = event.target as HTMLImageElement;
    const placeholder = img.nextElementSibling as HTMLElement;
    if (placeholder) {
      placeholder.style.display = 'none';
    }
  }
</script>

<button class="game-card" class:launching={isLaunching} onclick={handleClick}>
  <div class="card-art">
    <img 
      src={artUrl} 
      alt={game.name}
      onload={handleImageLoad}
      onerror={handleImageError}
    />
    <div class="art-placeholder">
      <span class="placeholder-icon">🎮</span>
    </div>
  </div>
  
  <div class="card-content">
    <h3 class="game-title">{game.name}</h3>
    
    <div class="game-meta">
      <span class="platform-badge" data-platform={instance.platform}>
        {instance.platform}
      </span>
      
      {#if instance.fileSize}
        <span class="size-badge">
          {formatFileSize(instance.fileSize)}
        </span>
      {/if}
    </div>
    
    {#if instance.installed}
      <span class="installed-indicator">
        <span class="check-icon">✓</span> Installed
      </span>
    {/if}
  </div>
</button>

<style>
  .game-card {
    display: flex;
    flex-direction: column;
    background: rgba(30, 41, 59, 0.8);
    border: 1px solid rgba(148, 163, 184, 0.2);
    border-radius: 12px;
    overflow: hidden;
    cursor: pointer;
    transition: all 0.2s ease;
    text-align: left;
    padding: 0;
    width: 100%;
    height: max-content;
  }
  
  .game-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 25px rgba(0, 0, 0, 0.3);
    border-color: rgba(148, 163, 184, 0.4);
  }

  .game-card.launching {
    box-shadow: 0 0 0 2px #60a5fa;
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% {
      box-shadow: 0 0 0 2px #60a5fa;
    }
    50% {
      box-shadow: 0 0 0 4px #60a5fa, 0 0 15px rgba(96, 165, 250, 0.5);
    }
  }
  
  .card-art {
    position: relative;
    width: 100%;
    aspect-ratio: 460 / 215;
    background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
    overflow: hidden;
  }
  
  .card-art img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    position: absolute;
    top: 0;
    left: 0;
  }
  
  .art-placeholder {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
  }
  
  .card-art img:not([style*="display: none"]) + .art-placeholder {
    display: none;
  }
  
  .placeholder-icon {
    font-size: 3rem;
    opacity: 0.5;
  }
  
  .card-content {
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .game-title {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: #f1f5f9;
    line-height: 1.3;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .game-meta {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    flex-wrap: wrap;
  }
  
  .platform-badge {
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 500;
    text-transform: uppercase;
    background: rgba(99, 102, 241, 0.2);
    color: #818cf8;
  }
  
  .platform-badge[data-platform="steam"] {
    background: rgba(59, 130, 246, 0.2);
    color: #60a5fa;
  }
  
  .size-badge {
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    color: #94a3b8;
    background: rgba(148, 163, 184, 0.1);
  }
  
  .installed-indicator {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.75rem;
    color: #4ade80;
    margin-top: 0.25rem;
  }
  
  .check-icon {
    font-weight: bold;
  }
</style>
