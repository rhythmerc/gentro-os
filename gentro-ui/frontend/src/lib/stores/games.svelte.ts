import type { GameWithInstance } from '../../../bindings/github.com/rhythmerc/gentro-ui/services/games/models/models.js';
import { GameFilter } from '../../../bindings/github.com/rhythmerc/gentro-ui/services/games/models/models.js';
import { GamesService } from '../../../bindings/github.com/rhythmerc/gentro-ui/services/games/index.js';

export interface GameStore {
	games: GameWithInstance[];
	loading: boolean;
	error: string | null;
}

export function createGamesStore() {
	let games = $state<GameWithInstance[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function loadGames(filter?: GameFilter) {
		loading = true;
		error = null;
		try {
			// Use provided filter or create default with Steam tool exclusion
			const gameFilter = filter ?? new GameFilter({
				installedOnly: false,
				sourceFilters: {
					steam: {
						excludeTools: true
					}
				}
			});
			const result = await GamesService.GetGames(gameFilter, null);
			games = result;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load games';
			console.error('Failed to load games:', err);
		} finally {
			loading = false;
		}
	}

	function getArtUrl(instanceId: string, artType: string = 'header'): string {
		return `/games/art/${instanceId}/${artType}`;
	}

	function formatFileSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
	}

	return {
		get games() { return games; },
		get loading() { return loading; },
		get error() { return error; },
		loadGames,
		getArtUrl,
		formatFileSize
	};
}

export const gamesStore = createGamesStore();
