'use client';

import React, { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import {
  Terminal,
  Play,
  Activity,
  CheckCircle,
  AlertTriangle,
  Cpu,
  Globe,
  Search,
  RefreshCw,
  BarChart2,
  BookOpen,
  Code,
  Check,
  Copy,
  ChevronRight,
  Server
} from 'lucide-react';

interface Script {
  name: string;
  displayName: string;
  path: string;
  code: string;
  readme: string;
  isProductionReady: boolean;
}

interface DashboardPortalProps {
  initialScripts: Script[];
  apiURL: string;
  grafanaURL: string;
}

export default function DashboardPortal({ initialScripts, apiURL, grafanaURL }: DashboardPortalProps) {
  const [scripts] = useState<Script[]>(initialScripts);
  const [activeScript, setActiveScript] = useState<Script | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedEnv, setSelectedEnv] = useState('dev');
  const [executing, setExecuting] = useState(false);
  const [activeTab, setActiveTab] = useState<'docs' | 'code' | 'telemetry'>('docs');

  // Terminal Output states
  const [terminalLogs, setTerminalLogs] = useState<string>('');
  const [execStatus, setExecStatus] = useState<'idle' | 'success' | 'failed'>('idle');
  const [exitCode, setExitCode] = useState<number | null>(null);
  const [executionID, setExecutionID] = useState<string | null>(null);

  // Copied state
  const [copied, setCopied] = useState(false);

  // Filter scripts based on search query (matches name, path, or readme content)
  const filteredScripts = scripts.filter((script) => {
    const query = searchQuery.toLowerCase();
    return (
      script.name.toLowerCase().includes(query) ||
      script.path.toLowerCase().includes(query) ||
      script.readme.toLowerCase().includes(query)
    );
  });

  const handleCopyCode = () => {
    if (!activeScript) return;
    navigator.clipboard.writeText(activeScript.code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleTriggerExecution = async () => {
    if (!activeScript) return;

    setExecuting(true);
    setTerminalLogs(`[SYSTEM] Initializing execution session...\n[SYSTEM] Fetching script ${activeScript.path} from branch main...\n[SYSTEM] Connecting to target environment: ${selectedEnv}...\n[SYSTEM] Triggering Go API at ${apiURL}/api/v1/execute...\n\n`);
    setExecStatus('idle');
    setExitCode(null);
    setExecutionID(null);

    try {
      const res = await fetch(`${apiURL}/api/v1/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          script_path: activeScript.path,
          branch: 'main',
          environment: selectedEnv,
        }),
      });

      const data = await res.json();

      if (!res.ok) {
        setExecStatus('failed');
        setTerminalLogs(
          (prev) =>
            prev +
            `[API ERROR] ${data.error || 'Server error occurred during execution'}\n` +
            (data.stderr ? `\n--- STDERR ---\n${data.stderr}` : '') +
            (data.stdout ? `\n--- STDOUT ---\n${data.stdout}` : '')
        );
        if (data.execution_id) setExecutionID(data.execution_id);
        if (data.exit_code !== undefined) setExitCode(data.exit_code);
        return;
      }

      setExecutionID(data.execution_id);
      setExitCode(data.exit_code);

      let logOutput = '';
      if (data.stdout) {
        logOutput += `--- STDOUT ---\n${data.stdout}\n`;
      }
      if (data.stderr) {
        logOutput += `--- STDERR ---\n${data.stderr}\n`;
      }

      setTerminalLogs(
        (prev) =>
          prev +
          logOutput +
          `\n[SYSTEM] Process completed.\n[SYSTEM] Execution ID: ${data.execution_id}\n[SYSTEM] Exit Code: ${data.exit_code}\n`
      );

      if (data.exit_code === 0) {
        setExecStatus('success');
      } else {
        setExecStatus('failed');
      }
    } catch (err: any) {
      setExecStatus('failed');
      setTerminalLogs((prev) => prev + `[CONNECTION ERROR] Failed to connect to Go server: ${err.message}\n`);
    } finally {
      setExecuting(false);
    }
  };

  return (
    <div className="flex h-screen bg-zinc-950 text-zinc-100 font-sans">
      {/* Sidebar */}
      <aside className="w-80 border-r border-zinc-800 flex flex-col bg-zinc-900/50 backdrop-blur-md">
        {/* Brand */}
        <div className="p-6 border-b border-zinc-800 flex items-center gap-3">
          <div className="bg-indigo-600 p-2 rounded-lg text-white">
            <Cpu size={20} className="animate-pulse" />
          </div>
          <div>
            <h1 className="font-bold text-lg tracking-wider bg-gradient-to-r from-white to-zinc-400 bg-clip-text text-transparent">
              COGNISHELL
            </h1>
            <p className="text-xs text-zinc-500 font-medium">Orchestration & Telemetry</p>
          </div>
        </div>

        {/* Search */}
        <div className="p-4 border-b border-zinc-800">
          <div className="relative">
            <Search className="absolute left-3 top-2.5 text-zinc-500" size={16} />
            <input
              type="text"
              placeholder="Search scripts, path, docs..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-zinc-950/80 border border-zinc-800 rounded-lg py-2 pl-10 pr-4 text-sm text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all"
            />
          </div>
        </div>

        {/* Script Catalog list */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          <div>
            <h2 className="text-xs font-bold text-zinc-500 uppercase tracking-widest px-2 mb-2">
              Script Catalog ({filteredScripts.length})
            </h2>
            <div className="space-y-1">
              {filteredScripts.map((script) => (
                <button
                  key={script.path}
                  onClick={() => {
                    setActiveScript(script);
                    setActiveTab('docs');
                    setTerminalLogs('');
                    setExecStatus('idle');
                    setExitCode(null);
                    setExecutionID(null);
                  }}
                  className={`w-full flex items-center justify-between p-3 rounded-lg text-left text-sm transition-all group ${
                    activeScript?.path === script.path
                      ? 'bg-indigo-600/10 border border-indigo-500/30 text-white font-medium shadow-md shadow-indigo-600/5'
                      : 'hover:bg-zinc-800/60 border border-transparent text-zinc-400 hover:text-zinc-200'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <Server size={16} className={activeScript?.path === script.path ? 'text-indigo-400' : 'text-zinc-500 group-hover:text-zinc-400'} />
                    <span className="truncate max-w-[170px]">{script.displayName}</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    {script.isProductionReady && (
                      <span className="text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-1.5 py-0.5 rounded-full font-medium">
                        Prod
                      </span>
                    )}
                    <ChevronRight size={14} className="opacity-0 group-hover:opacity-100 transition-opacity" />
                  </div>
                </button>
              ))}
              {filteredScripts.length === 0 && (
                <p className="text-sm text-zinc-600 text-center py-8">No scripts found</p>
              )}
            </div>
          </div>
        </div>

        {/* System Node status */}
        <div className="p-4 border-t border-zinc-800 bg-zinc-950/20">
          <button
            onClick={() => setActiveScript(null)}
            className={`w-full flex items-center gap-3 p-3 rounded-lg text-sm transition-all border ${
              activeScript === null
                ? 'bg-zinc-800 border-zinc-700 text-white'
                : 'hover:bg-zinc-900 border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            <Activity size={16} className="text-indigo-400" />
            <span className="font-medium">System Dashboard Overview</span>
          </button>
        </div>
      </aside>

      {/* Main Panel */}
      <main className="flex-1 flex flex-col bg-zinc-950 overflow-y-auto">
        {activeScript ? (
          /* Detailed Script View */
          <div className="flex-1 flex flex-col h-full">
            {/* Header */}
            <header className="p-6 border-b border-zinc-900 flex justify-between items-center bg-zinc-900/10">
              <div>
                <div className="flex items-center gap-3 mb-1">
                  <h2 className="text-xl font-bold text-white">{activeScript.displayName}</h2>
                  <span className="text-xs bg-zinc-800 border border-zinc-700 px-2 py-0.5 rounded-md text-zinc-400">
                    main branch
                  </span>
                </div>
                <p className="text-sm text-zinc-500 font-mono">{activeScript.path}</p>
              </div>

              {/* Execution Actions */}
              <div className="flex items-center gap-3">
                <div className="flex items-center bg-zinc-900 border border-zinc-800 rounded-lg p-1">
                  <span className="text-xs text-zinc-500 px-3 font-medium">Env:</span>
                  <select
                    value={selectedEnv}
                    onChange={(e) => setSelectedEnv(e.target.value)}
                    className="bg-zinc-950 border border-zinc-800 rounded px-2.5 py-1 text-xs font-semibold text-zinc-300 focus:outline-none focus:border-indigo-500"
                  >
                    <option value="dev">Development</option>
                    <option value="staging">Staging</option>
                    <option value="prod">Production</option>
                  </select>
                </div>

                <button
                  onClick={handleTriggerExecution}
                  disabled={executing}
                  className={`flex items-center gap-2 px-5 py-2 rounded-lg text-sm font-semibold shadow-lg shadow-indigo-600/10 transition-all ${
                    executing
                      ? 'bg-zinc-800 border border-zinc-700 text-zinc-500 cursor-not-allowed'
                      : 'bg-indigo-600 hover:bg-indigo-500 text-white hover:scale-[1.02] active:scale-[0.98]'
                  }`}
                >
                  {executing ? (
                    <RefreshCw className="animate-spin" size={16} />
                  ) : (
                    <Play fill="currentColor" size={14} />
                  )}
                  {executing ? 'Executing...' : 'Trigger Execution'}
                </button>
              </div>
            </header>

            {/* Tab Navigation */}
            <div className="flex border-b border-zinc-900 bg-zinc-900/10 px-6">
              <button
                onClick={() => setActiveTab('docs')}
                className={`py-3 px-4 border-b-2 text-sm font-medium transition-all flex items-center gap-2 ${
                  activeTab === 'docs'
                    ? 'border-indigo-500 text-indigo-400'
                    : 'border-transparent text-zinc-400 hover:text-zinc-200'
                }`}
              >
                <BookOpen size={16} />
                Documentation
              </button>
              <button
                onClick={() => setActiveTab('code')}
                className={`py-3 px-4 border-b-2 text-sm font-medium transition-all flex items-center gap-2 ${
                  activeTab === 'code'
                    ? 'border-indigo-500 text-indigo-400'
                    : 'border-transparent text-zinc-400 hover:text-zinc-200'
                }`}
              >
                <Code size={16} />
                Script Source
              </button>
              <button
                onClick={() => setActiveTab('telemetry')}
                className={`py-3 px-4 border-b-2 text-sm font-medium transition-all flex items-center gap-2 ${
                  activeTab === 'telemetry'
                    ? 'border-indigo-500 text-indigo-400'
                    : 'border-transparent text-zinc-400 hover:text-zinc-200'
                }`}
              >
                <BarChart2 size={16} />
                Grafana Telemetry
              </button>
            </div>

            {/* Tab Content */}
            <div className="flex-1 p-6 space-y-6 flex flex-col min-h-0 overflow-y-auto">
              <div className="flex-1 grid grid-cols-1 lg:grid-cols-2 gap-6 min-h-0">
                {/* Left Panel: Documentation / Code / Telemetry */}
                <div className="bg-zinc-900/30 border border-zinc-900 rounded-xl p-6 overflow-y-auto flex flex-col min-h-[300px]">
                  {activeTab === 'docs' && (
                    <div className="markdown-body flex-1">
                      <ReactMarkdown>{activeScript.readme}</ReactMarkdown>
                    </div>
                  )}

                  {activeTab === 'code' && (
                    <div className="flex flex-col flex-1 min-h-0">
                      <div className="flex justify-between items-center mb-3">
                        <span className="text-xs text-zinc-500 font-mono">{activeScript.name}.ps1</span>
                        <button
                          onClick={handleCopyCode}
                          className="flex items-center gap-1.5 text-xs text-zinc-400 hover:text-white transition-all bg-zinc-800 hover:bg-zinc-700 px-2.5 py-1 rounded"
                        >
                          {copied ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
                          {copied ? 'Copied' : 'Copy'}
                        </button>
                      </div>
                      <pre className="bg-zinc-950 p-4 border border-zinc-800 rounded-lg overflow-auto font-mono text-xs text-zinc-300 flex-1 whitespace-pre-wrap leading-relaxed">
                        <code>{activeScript.code}</code>
                      </pre>
                    </div>
                  )}

                  {activeTab === 'telemetry' && (
                    <div className="flex-1 flex flex-col gap-6">
                      <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                        <div>
                          <h3 className="font-semibold text-white">Live Grafana Dashboard Panels</h3>
                          <p className="text-xs text-zinc-500">Telemetry pulled from Prometheus database</p>
                        </div>
                        <span className="text-xs bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 px-2 py-0.5 rounded">
                          Integrated iframe
                        </span>
                      </div>

                      {/* Embedded Grafana iframe Success panel */}
                      <div className="border border-zinc-800 rounded-lg overflow-hidden h-[240px] bg-zinc-950">
                        <iframe
                          src={`${grafanaURL}/d-solo/cognishell-dashboard/cognishell-script-metrics?orgId=1&panelId=1&refresh=5s&theme=dark&var-script_name=${activeScript.path}`}
                          width="100%"
                          height="100%"
                          frameBorder="0"
                          title="Grafana Success Rates"
                          className="bg-zinc-950"
                        ></iframe>
                      </div>

                      {/* Embedded Grafana iframe Duration panel */}
                      <div className="border border-zinc-800 rounded-lg overflow-hidden h-[240px] bg-zinc-950">
                        <iframe
                          src={`${grafanaURL}/d-solo/cognishell-dashboard/cognishell-script-metrics?orgId=1&panelId=2&refresh=5s&theme=dark&var-script_name=${activeScript.path}`}
                          width="100%"
                          height="100%"
                          frameBorder="0"
                          title="Grafana Execution Times"
                          className="bg-zinc-950"
                        ></iframe>
                      </div>
                    </div>
                  )}
                </div>

                {/* Right Panel: Execution Console Terminal */}
                <div className="bg-zinc-950 border border-zinc-900 rounded-xl flex flex-col overflow-hidden h-full min-h-[400px]">
                  {/* Console Header */}
                  <div className="bg-zinc-900/60 px-4 py-3 border-b border-zinc-900 flex justify-between items-center">
                    <div className="flex items-center gap-2">
                      <Terminal size={15} className="text-emerald-400" />
                      <span className="text-xs font-bold text-zinc-400 tracking-wide uppercase">Execution Console</span>
                    </div>

                    <div className="flex items-center gap-2">
                      {executionID && (
                        <span className="text-[10px] font-mono text-zinc-500">ID: {executionID.slice(0, 8)}...</span>
                      )}
                      {execStatus === 'success' && (
                        <span className="flex items-center gap-1 text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-2 py-0.5 rounded font-bold uppercase">
                          <CheckCircle size={10} /> Success
                        </span>
                      )}
                      {execStatus === 'failed' && (
                        <span className="flex items-center gap-1 text-[10px] bg-rose-500/10 text-rose-400 border border-rose-500/20 px-2 py-0.5 rounded font-bold uppercase">
                          <AlertTriangle size={10} /> Failed
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Console Text Area */}
                  <div className="flex-1 p-5 font-mono text-xs text-emerald-400 overflow-y-auto leading-relaxed select-text space-y-1">
                    {terminalLogs ? (
                      <pre className="whitespace-pre-wrap select-text">{terminalLogs}</pre>
                    ) : (
                      <p className="text-zinc-600 select-none">
                        Ready. Select target environment and click "Trigger Execution" above to run this automation.
                      </p>
                    )}
                    {executing && (
                      <span className="inline-block w-2 h-4 bg-emerald-400 terminal-cursor ml-1 align-middle"></span>
                    )}
                  </div>

                  {/* Console Footer Status */}
                  <div className="bg-zinc-900/30 px-4 py-2.5 border-t border-zinc-900 flex justify-between text-[11px] text-zinc-500">
                    <span>Target Node: Windows Core via pwsh</span>
                    <span>Exit Code: {exitCode !== null ? exitCode : '-'}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        ) : (
          /* General Dashboard Overview */
          <div className="p-8 max-w-6xl mx-auto w-full space-y-8">
            {/* Greeting */}
            <div>
              <h2 className="text-3xl font-extrabold text-white tracking-tight mb-2">System Dashboard</h2>
              <p className="text-sm text-zinc-400">
                Welcome to the CogniShell Central Management Panel. Monitor orchestrations and execute production automations.
              </p>
            </div>

            {/* Metric cards grid */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
              <div className="bg-zinc-900 border border-zinc-800/80 p-5 rounded-xl flex items-center justify-between group hover:border-zinc-700 transition-all duration-300">
                <div>
                  <p className="text-xs font-bold text-zinc-500 uppercase tracking-widest mb-1">Catalog Scripts</p>
                  <h3 className="text-2xl font-extrabold text-white">{scripts.length}</h3>
                </div>
                <div className="bg-indigo-500/10 text-indigo-400 p-3 rounded-lg">
                  <Server size={20} />
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800/80 p-5 rounded-xl flex items-center justify-between group hover:border-zinc-700 transition-all duration-300">
                <div>
                  <p className="text-xs font-bold text-zinc-500 uppercase tracking-widest mb-1">Success Ratio</p>
                  <h3 className="text-2xl font-extrabold text-emerald-400">98.2%</h3>
                </div>
                <div className="bg-emerald-500/10 text-emerald-400 p-3 rounded-lg">
                  <CheckCircle size={20} />
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800/80 p-5 rounded-xl flex items-center justify-between group hover:border-zinc-700 transition-all duration-300">
                <div>
                  <p className="text-xs font-bold text-zinc-500 uppercase tracking-widest mb-1">Avg Run Duration</p>
                  <h3 className="text-2xl font-extrabold text-white">1.15s</h3>
                </div>
                <div className="bg-amber-500/10 text-amber-400 p-3 rounded-lg">
                  <Activity size={20} />
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800/80 p-5 rounded-xl flex items-center justify-between group hover:border-zinc-700 transition-all duration-300">
                <div>
                  <p className="text-xs font-bold text-zinc-500 uppercase tracking-widest mb-1">Loki Status</p>
                  <h3 className="text-2xl font-extrabold text-white">ACTIVE</h3>
                </div>
                <div className="bg-emerald-500/10 text-emerald-400 p-3 rounded-lg">
                  <Globe size={20} />
                </div>
              </div>
            </div>

            {/* Secondary layout sections */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              {/* Active Catalog List */}
              <div className="lg:col-span-2 bg-zinc-900 border border-zinc-800/80 rounded-xl p-6 space-y-4">
                <h3 className="text-base font-bold text-white tracking-wide border-b border-zinc-800 pb-3">Available Automation Scripts</h3>
                <div className="divide-y divide-zinc-800/60">
                  {scripts.map((script) => (
                    <div key={script.path} className="py-4 flex justify-between items-center first:pt-0 last:pb-0">
                      <div>
                        <h4 className="font-semibold text-white text-sm">{script.displayName}</h4>
                        <p className="text-xs text-zinc-500 font-mono">{script.path}</p>
                      </div>
                      <button
                        onClick={() => {
                          setActiveScript(script);
                          setActiveTab('docs');
                        }}
                        className="text-xs text-indigo-400 hover:text-indigo-300 font-bold flex items-center gap-1 transition-all"
                      >
                        Inspect Script <ChevronRight size={14} />
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              {/* Status details */}
              <div className="bg-zinc-900 border border-zinc-800/80 rounded-xl p-6 space-y-4">
                <h3 className="text-base font-bold text-white tracking-wide border-b border-zinc-800 pb-3">Orchestrator Node Info</h3>
                <div className="space-y-4 text-sm">
                  <div className="flex justify-between border-b border-zinc-800/40 pb-2">
                    <span className="text-zinc-500">Go Backend</span>
                    <span className="font-mono text-zinc-300">{apiURL}</span>
                  </div>
                  <div className="flex justify-between border-b border-zinc-800/40 pb-2">
                    <span className="text-zinc-500">Grafana Telemetry</span>
                    <span className="font-mono text-zinc-300">{grafanaURL}</span>
                  </div>
                  <div className="flex justify-between border-b border-zinc-800/40 pb-2">
                    <span className="text-zinc-500">Primary VM IP</span>
                    <span className="font-mono text-zinc-300">10.0.0.11</span>
                  </div>
                  <div className="flex justify-between pb-1">
                    <span className="text-zinc-500">Vault Secrets</span>
                    <span className="text-emerald-400 flex items-center gap-1 font-semibold">
                      Connected <Check size={14} />
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
