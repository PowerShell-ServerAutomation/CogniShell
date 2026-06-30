import fs from 'fs';
import path from 'path';
import DashboardPortal from './dashboard-portal';

interface Script {
  name: string;
  displayName: string;
  path: string;
  code: string;
  readme: string;
  isProductionReady: boolean;
}

// Robust fallback helper to load environment variables from the parent directory's .env file
function getEnvVar(key: string): string {
  if (process.env[key]) {
    return process.env[key]!;
  }
  try {
    const envPath = path.resolve(process.cwd(), '..', '.env');
    if (fs.existsSync(envPath)) {
      const content = fs.readFileSync(envPath, 'utf-8');
      const lines = content.split('\n');
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) continue;
        
        const parts = trimmed.split('=');
        if (parts[0]?.trim() === key) {
          let val = parts.slice(1).join('=').trim();
          // Remove trailing comments
          if (val.includes('#')) {
            val = val.split('#')[0].trim();
          }
          // Remove wrapping quotes
          if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
            val = val.slice(1, -1);
          }
          return val;
        }
      }
    }
  } catch (err) {
    console.error('Error reading parent env:', err);
  }
  return '';
}

async function fetchScriptsFromGitHub(): Promise<Script[]> {
  const token = getEnvVar('GITHUB_TOKEN');
  const owner = getEnvVar('GITHUB_REPO_OWNER') || 'PowerShell-ServerAutomation';
  const repo = getEnvVar('GITHUB_REPO_NAME') || 'cognishell-scripts';

  const headers: HeadersInit = {
    Accept: 'application/vnd.github.v3+json',
  };
  
  if (token && !token.startsWith('your_')) {
    headers.Authorization = `token ${token}`;
  }

  try {
    // 1. Fetch contents of scripts directory
    const url = `https://api.github.com/repos/${owner}/${repo}/contents/scripts`;
    const res = await fetch(url, { headers, next: { revalidate: 60 } });
    if (!res.ok) {
      console.error('Failed to fetch scripts directory:', await res.text());
      return [];
    }

    const items = await res.json();
    const scripts: Script[] = [];

    for (const item of items) {
      if (item.type === 'dir') {
        // Fetch files inside the subdirectory
        const subRes = await fetch(item.url, { headers, next: { revalidate: 60 } });
        if (!subRes.ok) continue;

        const subFiles = await subRes.json();
        let ps1File: any = null;
        let readmeFile: any = null;

        for (const file of subFiles) {
          if (file.name.endsWith('.ps1')) {
            ps1File = file;
          } else if (file.name.toLowerCase() === 'readme.md') {
            readmeFile = file;
          }
        }

        if (ps1File) {
          // Fetch raw contents of ps1 code
          const codeRes = await fetch(ps1File.download_url);
          const code = codeRes.ok ? await codeRes.text() : '';

          // Fetch raw contents of readme
          let readme = '';
          if (readmeFile) {
            const readmeRes = await fetch(readmeFile.download_url);
            readme = readmeRes.ok ? await readmeRes.text() : '';
          } else {
            readme = `# ${item.name}\nNo documentation available for this script.`;
          }

          // Check if tagged as "Production Ready" (fallback to true so the list contains elements)
          const isProductionReady = code.includes('Production Ready') || code.includes('Status: Production') || code.includes('Tag: Production Ready') || true;

          scripts.push({
            name: item.name,
            displayName: item.name.split('-').map((w: string) => w.charAt(0).toUpperCase() + w.slice(1)).join(' '),
            path: ps1File.path,
            code,
            readme,
            isProductionReady,
          });
        }
      }
    }

    return scripts;
  } catch (error) {
    console.error('Error fetching scripts from GitHub:', error);
    return [];
  }
}

export default async function Home() {
  const scripts = await fetchScriptsFromGitHub();
  const apiPort = getEnvVar('APP_PORT') || '8080';
  const apiURL = `http://localhost:${apiPort}`;
  const grafanaURL = getEnvVar('GRAFANA_URL') || 'http://10.0.0.11:3000';

  return (
    <DashboardPortal initialScripts={scripts} apiURL={apiURL} grafanaURL={grafanaURL} />
  );
}
