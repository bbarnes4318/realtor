"use client";

import { useEffect, useState, useRef, use } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

interface JobFilters {
	state: string;
	city: string;
	zip: string;
	brokerage: string;
	agent_name: string;
	area_served: string;
}

interface Job {
	id: string;
	name: string;
	status: string;
	max_agents_limit: number;
	concurrency: number;
	throttle_request_limit: number;
	save_raw_agents: boolean;
	db_mode: string;
	filters: JobFilters | null;
	total_estimated_requests: number;
	completed_requests: number;
	failed_requests: number;
	agents_saved: number;
	started_at: string | null;
	completed_at: string | null;
	error_message: string | null;
}

interface LogLine {
	timestamp: string;
	level: string;
	message: string;
}

interface Agent {
	id: string;
	name: string;
	title: string;
	email: string;
	rating: number;
	city: string;
	state: string;
	zip: string;
	brokerage: string;
	office_name: string;
}

export default function JobDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const jobID = resolvedParams.id;
  const router = useRouter();
  
  const [job, setJob] = useState<Job | null>(null);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const consoleEndRef = useRef<HTMLDivElement | null>(null);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  const fetchJobDetails = async () => {
    try {
      // 1. Fetch Job
      const jobRes = await fetch(`${API_BASE}/api/jobs/${jobID}`);
      if (!jobRes.ok) throw new Error("Failed to fetch job details");
      const jobData = await jobRes.json();
      setJob(jobData);

      // 2. Fetch Logs
      const logsRes = await fetch(`${API_BASE}/api/jobs/${jobID}/logs`);
      if (logsRes.ok) {
        const logsData = await logsRes.json();
        setLogs(logsData);
      }

      // 3. Fetch Scraped Agents
      const agentsRes = await fetch(`${API_BASE}/api/agents?job_id=${jobID}&limit=10`);
      if (agentsRes.ok) {
        const agentsData = await agentsRes.json();
        setAgents(agentsData.agents || []);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchJobDetails();

    // Poll logs & progress every 2 seconds if the job is active/running/paused
    const interval = setInterval(() => {
      fetchJobDetails();
    }, 2000);

    return () => clearInterval(interval);
  }, [jobID, API_BASE]);

  // Auto-scroll logs console to bottom on new lines
  useEffect(() => {
    if (consoleEndRef.current) {
      consoleEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [logs]);

  const handleAction = async (action: string) => {
    try {
      const res = await fetch(`${API_BASE}/api/jobs/${jobID}/${action}`, {
        method: "POST",
      });
      if (!res.ok) throw new Error(`Failed to ${action} job`);
      fetchJobDetails();
    } catch (err: any) {
      alert(`Action error: ${err.message}`);
    }
  };

  if (loading && !job) {
    return (
      <div className="flex-1 flex items-center justify-center p-12">
        <div className="text-slate-400 flex flex-col items-center gap-3">
          <div className="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
          <p className="font-medium">Loading job details...</p>
        </div>
      </div>
    );
  }

  if (!job) {
    return (
      <div className="p-8 max-w-4xl mx-auto text-center space-y-4">
        <p className="text-xl font-bold text-red-400">⚠️ Scrape Job Not Found</p>
        <p className="text-slate-400">The scrape run ID {jobID} was not found in the database.</p>
        <Link href="/jobs" className="text-blue-400 hover:underline">
          Return to Jobs List
        </Link>
      </div>
    );
  }

  const percent = job.total_estimated_requests > 0
    ? Math.round((job.completed_requests / job.total_estimated_requests) * 100)
    : 0;

  return (
    <div className="p-8 max-w-7xl mx-auto w-full space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-800 pb-6">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-3xl font-bold tracking-tight">{job.name}</h2>
            <span
              className={`px-3 py-1 rounded-full text-xs font-bold uppercase ${
                job.status === "completed"
                  ? "bg-green-950/80 text-green-300 border border-green-800"
                  : job.status === "running"
                  ? "bg-blue-950/80 text-blue-300 border border-blue-800 animate-pulse"
                  : job.status === "paused"
                  ? "bg-yellow-950/80 text-yellow-300 border border-yellow-800"
                  : job.status === "failed"
                  ? "bg-red-950/80 text-red-300 border border-red-800"
                  : "bg-slate-800 text-slate-300 border border-slate-700"
              }`}
            >
              {job.status}
            </span>
          </div>
          <p className="text-slate-500 font-mono text-xs mt-1.5">Job ID: {job.id}</p>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-2">
          {job.status === "running" && (
            <button
              onClick={() => handleAction("pause")}
              className="bg-yellow-600 hover:bg-yellow-500 text-white font-semibold text-sm px-4 py-2 rounded-lg transition-colors"
            >
              ⏸️ Pause Job
            </button>
          )}

          {job.status === "paused" && (
            <button
              onClick={() => handleAction("resume")}
              className="bg-green-600 hover:bg-green-500 text-white font-semibold text-sm px-4 py-2 rounded-lg transition-colors"
            >
              ▶️ Resume Job
            </button>
          )}

          {(job.status === "running" || job.status === "paused") && (
            <button
              onClick={() => handleAction("cancel")}
              className="bg-red-600 hover:bg-red-500 text-white font-semibold text-sm px-4 py-2 rounded-lg transition-colors"
            >
              🛑 Cancel Job
            </button>
          )}

          <a
            href={`${API_BASE}/api/export/jobs/${job.id}.csv`}
            download
            className="bg-slate-950 border border-slate-800 hover:bg-slate-900 text-slate-200 font-semibold text-sm px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
          >
            📥 Export CSV
          </a>
        </div>
      </div>

      {/* Progress Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 space-y-2">
          <span className="text-slate-400 text-xs font-semibold block">PROGRESS PERCENT</span>
          <span className="text-3xl font-bold font-mono text-blue-400 block">{percent}%</span>
          <div className="w-full bg-slate-850 h-1.5 rounded-full overflow-hidden">
            <div className="bg-blue-500 h-1.5 rounded-full" style={{ width: `${percent}%` }}></div>
          </div>
        </div>

        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 space-y-2">
          <span className="text-slate-400 text-xs font-semibold block">REQUESTS COMPLETED</span>
          <span className="text-3xl font-bold font-mono text-green-400 block">
            {job.completed_requests} <span className="text-xs text-slate-500">/ {job.total_estimated_requests}</span>
          </span>
          <span className="text-xs text-slate-500 block">Total pages of 20 results</span>
        </div>

        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 space-y-2">
          <span className="text-slate-400 text-xs font-semibold block">FAILED REQUESTS</span>
          <span className="text-3xl font-bold font-mono text-red-400 block">{job.failed_requests}</span>
          <span className="text-xs text-slate-500 block">Pages that failed network retry limit</span>
        </div>

        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 space-y-2">
          <span className="text-slate-400 text-xs font-semibold block">AGENTS SAVED</span>
          <span className="text-3xl font-bold font-mono text-teal-400 block">{job.agents_saved}</span>
          <span className="text-xs text-slate-500 block">Records added/updated in DB</span>
        </div>
      </div>

      {/* Error Banner if failed */}
      {job.error_message && (
        <div className="bg-red-950/40 border border-red-800 text-red-200 p-5 rounded-xl space-y-1">
          <p className="font-semibold">⚠️ Job Failure Error Message</p>
          <p className="text-sm font-mono">{job.error_message}</p>
        </div>
      )}

      {/* Main Layout Grid (Logs + Config) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Side: Console Logs (2 cols) */}
        <div className="lg:col-span-2 bg-slate-950 rounded-xl border border-slate-800 overflow-hidden flex flex-col h-[500px]">
          <div className="bg-slate-900 px-5 py-3.5 border-b border-slate-800 flex items-center justify-between">
            <h3 className="font-semibold text-sm">📺 Live Logs Console</h3>
            <span className="text-slate-500 text-xs font-mono">Real-time updates</span>
          </div>

          <div className="flex-1 p-5 overflow-y-auto font-mono text-xs text-slate-300 space-y-1 bg-black">
            {logs.length === 0 ? (
              <div className="text-slate-600 italic">Console idle. Awaiting log stream...</div>
            ) : (
              logs.map((log, idx) => {
                const color =
                  log.level === "ERROR"
                    ? "text-red-400"
                    : log.level === "WARN"
                    ? "text-yellow-400"
                    : log.level === "DEBUG"
                    ? "text-slate-600"
                    : "text-green-400";

                return (
                  <div key={idx} className="flex gap-4">
                    <span className="text-slate-600 select-none">
                      {new Date(log.timestamp).toLocaleTimeString()}
                    </span>
                    <span className={`${color} font-bold select-none`}>[{log.level}]</span>
                    <span className="break-all whitespace-pre-wrap">{log.message}</span>
                  </div>
                );
              })
            )}
            <div ref={consoleEndRef} />
          </div>
        </div>

        {/* Right Side: Config summary (1 col) */}
        <div className="bg-slate-950 rounded-xl border border-slate-800 p-6 space-y-6 h-fit">
          <h3 className="font-semibold border-b border-slate-800 pb-3">📁 Scraper Scope</h3>

          <div className="space-y-4 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-400">Concurrency Threads</span>
              <span className="font-mono text-slate-200">{job.concurrency}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Throttle Limit</span>
              <span className="font-mono text-slate-200">{job.throttle_request_limit} reqs</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Save Raw Payloads</span>
              <span className="text-slate-200">{job.save_raw_agents ? "Enabled" : "Disabled"}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">DB Mode</span>
              <span className="font-mono uppercase text-slate-200">{job.db_mode}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Start Time</span>
              <span className="text-slate-300 text-xs">
                {job.started_at ? new Date(job.started_at).toLocaleString() : "Pending"}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Completed Time</span>
              <span className="text-slate-300 text-xs">
                {job.completed_at ? new Date(job.completed_at).toLocaleString() : "Running..."}
              </span>
            </div>
          </div>

          <div className="border-t border-slate-800 pt-4 space-y-4">
            <h4 className="font-semibold text-xs text-slate-400 uppercase tracking-wider">🎯 Scrape Filters</h4>
            {job.filters ? (
              <div className="space-y-3.5 text-sm">
                {job.filters.zip && (
                  <div className="flex justify-between">
                    <span className="text-slate-500">Zip Code</span>
                    <span className="font-mono text-slate-200">{job.filters.zip}</span>
                  </div>
                )}
                {job.filters.agent_name && (
                  <div className="flex justify-between">
                    <span className="text-slate-500">Agent Name</span>
                    <span className="text-slate-200">{job.filters.agent_name}</span>
                  </div>
                )}
                {job.filters.city && (
                  <div className="flex justify-between">
                    <span className="text-slate-500">City</span>
                    <span className="text-slate-200">{job.filters.city}</span>
                  </div>
                )}
                {job.filters.state && (
                  <div className="flex justify-between">
                    <span className="text-slate-500">State</span>
                    <span className="font-mono text-slate-200">{job.filters.state}</span>
                  </div>
                )}
                {job.filters.brokerage && (
                  <div className="flex justify-between">
                    <span className="text-slate-500">Brokerage</span>
                    <span className="text-slate-200">{job.filters.brokerage}</span>
                  </div>
                )}
                {job.filters.area_served && (
                  <div className="flex justify-between">
                    <span className="text-slate-500">Area Served</span>
                    <span className="text-slate-200">{job.filters.area_served}</span>
                  </div>
                )}
                {!job.filters.zip && !job.filters.agent_name && !job.filters.city && !job.filters.state && !job.filters.brokerage && !job.filters.area_served && (
                  <span className="text-slate-500 italic text-xs block">No filters (Scraping nationwide)</span>
                )}
              </div>
            ) : (
              <span className="text-slate-500 italic text-xs block">No filters (Scraping nationwide)</span>
            )}
          </div>
        </div>
      </div>

      {/* Scraped Agents Table */}
      <div className="bg-slate-950 rounded-xl border border-slate-800 p-6 space-y-4">
        <h3 className="text-lg font-semibold">👤 Recently Scraped Agents in this Run</h3>

        {agents.length === 0 ? (
          <div className="text-center py-8 text-slate-650 text-sm italic">
            No agents saved yet by this run. Check console logs for active inserts.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-sm">
              <thead>
                <tr className="border-b border-slate-800 text-slate-450">
                  <th className="pb-3 font-semibold">Agent Name</th>
                  <th className="pb-3 font-semibold">Title</th>
                  <th className="pb-3 font-semibold">Brokerage</th>
                  <th className="pb-3 font-semibold">Office Name</th>
                  <th className="pb-3 font-semibold">City/State</th>
                  <th className="pb-3 font-semibold text-right">Detail</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-900">
                {agents.map((agent) => (
                  <tr key={agent.id} className="hover:bg-slate-900/40">
                    <td className="py-3 font-medium text-slate-200">{agent.name}</td>
                    <td className="py-3 text-slate-400">{agent.title || "Sales Agent"}</td>
                    <td className="py-3 text-slate-300">{agent.brokerage || "Independent"}</td>
                    <td className="py-3 text-slate-400">{agent.office_name || "N/A"}</td>
                    <td className="py-3 text-slate-400 font-mono text-xs">
                      {agent.city}, {agent.state}
                    </td>
                    <td className="py-3 text-right">
                      <Link
                        href={`/agents?search=${encodeURIComponent(agent.name)}`}
                        className="text-blue-400 hover:text-blue-300 text-xs font-semibold"
                      >
                        Profile →
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
