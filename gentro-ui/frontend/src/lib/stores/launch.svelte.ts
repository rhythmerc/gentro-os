import { Events } from "@wailsio/runtime";
import { GamesService } from '../../../bindings/github.com/rhythmerc/gentro-ui/services/games/index.js';
import { LaunchStatus, type LaunchStatusUpdate } from '../../../bindings/github.com/rhythmerc/gentro-ui/services/games/models/models.js';
import { gamesStore } from './games.svelte.js';

// WailsEvent type from runtime
type WailsEvent<T = string> = {
  name: T;
  data: any;
  sender?: string;
};

export interface ActiveLaunch {
  instanceId: string;
  gameId: string;
  gameName: string;
  status: LaunchStatus;
}

export function createLaunchStore() {
  // Use plain object for proper Svelte 5 reactivity
  let activeLaunches = $state<Record<string, ActiveLaunch>>({});

  // Use $derived for computed values - ensures reactivity
  const activeLaunchesList = $derived(Object.values(activeLaunches));
  const hasActiveLaunches = $derived(Object.keys(activeLaunches).length > 0);

  // Listen to launch events
  Events.On("launchStatusUpdate", (ev: WailsEvent<"launchStatusUpdate">) => {
    console.log("[LaunchStore] Received event:", ev.name, ev.data);
    const update = ev.data as LaunchStatusUpdate;
    
    // Create new object to trigger reactivity
    const newLaunches = { ...activeLaunches };
    const existing = newLaunches[update.instanceId];

    console.log("[LaunchStore] Processing status:", update.status, "for instance:", update.instanceId);

    switch (update.status) {
      case LaunchStatus.LaunchStatusLaunching: {
        // Look up game name from games store
        const gameWithInstance = gamesStore.games.find(
          g => g.instance.id === update.instanceId
        );
        const gameName = gameWithInstance?.game.name ?? "Unknown Game";

        newLaunches[update.instanceId] = {
          instanceId: update.instanceId,
          gameId: update.gameId,
          gameName: gameName,
          status: LaunchStatus.LaunchStatusLaunching
        };
        console.log("[LaunchStore] Added launching game:", gameName);
        break;
      }
      case LaunchStatus.LaunchStatusRunning: {
        if (existing) {
          newLaunches[update.instanceId] = {
            ...existing,
            status: LaunchStatus.LaunchStatusRunning
          };
          console.log("[LaunchStore] Game now running:", existing.gameName);
        }
        break;
      }
      case LaunchStatus.LaunchStatusStopped:
      case LaunchStatus.LaunchStatusFailed: {
        if (existing) {
          console.log("[LaunchStore] Removing game:", existing.gameName, "status:", update.status);
        }
        delete newLaunches[update.instanceId];
        break;
      }
    }
    
    // Reassign to trigger reactivity
    activeLaunches = newLaunches;
    console.log("[LaunchStore] Updated active launches, count:", Object.keys(newLaunches).length);
  });

  async function launchGame(instanceId: string) {
    console.log("[LaunchStore] launchGame called for:", instanceId);
    try {
      await GamesService.Launch(instanceId);
      console.log("[LaunchStore] GamesService.Launch returned successfully");
    } catch (err) {
      console.error("[LaunchStore] Failed to launch game:", err);
      // Remove from active launches if launch failed immediately
      const newLaunches = { ...activeLaunches };
      delete newLaunches[instanceId];
      activeLaunches = newLaunches;
    }
  }

  function isLaunching(instanceId: string): boolean {
    const launch = activeLaunches[instanceId];
    return launch !== undefined && launch.status === LaunchStatus.LaunchStatusLaunching;
  }

  return {
    get activeLaunches() { return activeLaunchesList; },
    get hasActiveLaunches() { return hasActiveLaunches; },
    launchGame,
    isLaunching
  };
}

export const launchStore = createLaunchStore();
