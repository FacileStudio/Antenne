import type { CliCommand } from '@facile/muse';

export const CLI_RELEASES = 'https://github.com/FacileStudio/antenne-cli/releases';

export const CLI_COMMANDS: CliCommand[] = [
	{ label: 'curl', command: 'curl -fsSL https://raw.githubusercontent.com/FacileStudio/antenne-cli/main/install.sh | bash' },
	{ label: 'facile', command: 'facile install antenne' },
	{ label: 'Homebrew', command: 'brew install --cask FacileStudio/tap/antenne' }
];
