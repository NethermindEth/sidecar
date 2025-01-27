// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://layr-labs.github.io',
	integrations: [
		starlight({
			title: 'EigenLayer Sidecar',
			social: {
				github: 'https://github.com/Layr-Labs/sidecar',
			},
			sidebar: [
				{
					label: 'About',
					items: [
						{ slug: 'about/overview', label: 'What is the Sidecar?' },
					]
				},
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
					items: []
				},
				{
					label: 'Sidecar API',
					items: []
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
