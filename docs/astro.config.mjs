// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'EigenLayer Sidecar',
			social: {
				github: 'https://github.com/Layr-Labs/sidecar',
			},
			sidebar: [
				{
					label: 'Running the Sidecar',
					items: [
						{ slug: 'running/getting-started', label: 'Getting Started' },
						{ slug: 'running/advanced-postgres', label: 'Advanced PostgreSQL Config' },
						{ slug: 'running/docker-compose', label: 'Running with Docker Compose' },
						{ slug: 'running/kubernetes', label: 'Running on Kubernetes' },

					],
				},
				{
					label: 'Rewards Calculations',
					autogenerate: { directory: 'reference' },
				},
				{
					label: 'Sidecar API',
					autogenerate: { directory: 'reference' },
				},
				{
					label: 'Contributing',
					items: [
						{ slug: 'development/building', label: 'Building' },
						{ slug: 'development/testing', label: 'Testing' },
						{ slug: 'development/extended-tests', label: 'Extended Tests' },
					]
				}
			],
		}),
	],
});
