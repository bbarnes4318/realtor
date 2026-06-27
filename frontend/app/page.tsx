"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

interface Stats {
	total_agents: number;
	total_phones: number;
	total_offices: number;
	total_brokerages: number;
	total_jobs: number;
	active_jobs: number;
	failed_jobs: number;
	last_run_date: string;
}

interface Job {
	id: string;
	name: string;
	status: string;
	total_estimated_requests: number;
	completed_requests: number;
	failed_requests: number;
	agents_saved: number;
	started_at: string | null;
	completed_at: string | null;
}

export default function OverviewPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [recentJobs, setRecentJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        // Fetch Stats
        const statsRes = await fetch(`${API_BASE}/api/stats`);
        if (!statsRes.ok) throw new Error("Failed to fetch dashboard stats");
        const statsData = await statsRes.json();
        setStats(statsData);

        // Fetch Jobs
        const jobsRes = await fetch(`${API_BASE}/api/jobs`);
        if (!jobsRes.ok) throw new Error("Failed to fetch jobs");
        const jobsData = await jobsRes.json();
        setRecentJobs(jobsData.slice(0, 5)); // show top 5 recent jobs
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [API_BASE]);

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center p-12">
        <div className="text-slate-400 flex flex-col items-center gap-3">
          <div className="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
          <p className="font-medium">Loading Dashboard statistics...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8 max-w-4xl mx-auto">
        <div className="bg-red-950/50 border border-red-800 text-red-200 p-6 rounded-xl flex flex-col gap-3">
          <p className="font-semibold text-lg">⚠️ Connection Failure</p>
          <p className="text-sm">Could not connect to the Go API backend server at {API_BASE}.</p>
          <p className="text-xs text-red-300 font-mono">Error: {error}</p>
          <div className="mt-2 text-xs text-slate-400">
            Ensure the Go server is running by executing:
            <code className="block bg-slate-950 p-2 rounded mt-1 font-mono text-slate-200">go run . --server</code>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8 max-w-7xl mx-auto w-full space-y-8">
      {/* Header */}
      <div>
        <h2 className="text-3xl font-bold tracking-tight">Overview Dashboard</h2>
        <p className="text-slate-400 mt-1">Real-time database metrics and scraper job control.</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Total Agents */}
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 flex flex-col gap-1">
          <span className="text-slate-400 text-sm font-medium">Total Agents Scraped</span>
          <span className="text-3xl font-bold text-blue-400 mt-2 font-mono">
            {stats?.total_agents.toLocaleString() || 0}
          </span>
        </div>

        {/* Total Phones */}
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 flex flex-col gap-1">
          <span className="text-slate-400 text-sm font-medium">Total Phones Captured</span>
          <span className="text-3xl font-bold text-teal-400 mt-2 font-mono">
            {stats?.total_phones.toLocaleString() || 0}
          </span>
        </div>

        {/* Total Offices */}
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 flex flex-col gap-1">
          <span className="text-slate-400 text-sm font-medium">Registered Offices</span>
          <span className="text-3xl font-bold text-indigo-400 mt-2 font-mono">
            {stats?.total_offices.toLocaleString() || 0}
          </span>
        </div>

        {/* Active Jobs */}
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 flex flex-col gap-1">
          <span className="text-slate-400 text-sm font-medium">Active Scrapes</span>
          <span className="text-3xl font-bold text-green-400 mt-2 font-mono">
            {stats?.active_jobs || 0} <span className="text-xs text-slate-500 font-normal">running</span>
          </span>
        </div>
      </div>

      {/* Stats Grid Rows 2 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 flex items-center justify-between">
          <div>
            <span className="text-slate-400 text-sm font-medium block">Total Scrape Runs</span>
            <span className="text-2xl font-bold mt-1 block font-mono">{stats?.total_jobs || 0}</span>
          </div>
          <span className="text-2xl">⚙️</span>
        </div>
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 flex items-center justify-between">
          <div>
            <span className="text-slate-400 text-sm font-medium block">Failed Runs</span>
            <span className="text-2xl font-bold mt-1 text-red-400 block font-mono">{stats?.failed_jobs || 0}</span>
          </div>
          <span className="text-2xl">⚠️</span>
        </div>
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 flex items-center justify-between">
          <div>
            <span className="text-slate-400 text-sm font-medium block">Last Run Date</span>
            <span className="text-sm font-medium mt-2 block font-mono text-slate-300">
              {stats?.last_run_date ? new Date(stats.last_run_date).toLocaleDateString() : "Never"}
            </span>
          </div>
          <span className="text-2xl">📅</span>
        </div>
      </div>

      {/* Recent Jobs Panel */}
      <div className="bg-slate-950 rounded-xl border border-slate-800 p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">Recent Scrape Runs</h3>
          <Link href="/jobs" className="text-blue-400 hover:text-blue-300 text-sm font-medium">
            View All Jobs →
          </Link>
        </div>

        {recentJobs.length === 0 ? (
          <div className="text-center py-8 text-slate-500 text-sm border-2 border-dashed border-slate-800 rounded-lg">
            No scrape jobs found. Head to "New Scrape Job" to start.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-sm">
              <thead>
                <tr className="border-b border-slate-800 text-slate-400">
                  <th className="pb-3 font-semibold">Job Name</th>
                  <th className="pb-3 font-semibold">Status</th>
                  <th className="pb-3 font-semibold">Progress</th>
                  <th className="pb-3 font-semibold">Agents Saved</th>
                  <th className="pb-3 font-semibold">Start Time</th>
                  <th className="pb-3 font-semibold text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-900">
                {recentJobs.map((job) => {
                  const percent = job.total_estimated_requests > 0
                    ? Math.round((job.completed_requests / job.total_estimated_requests) * 100)
                    : 0;

                  return (
                    <tr key={job.id} className="hover:bg-slate-900/40">
                      <td className="py-3.5 font-medium">{job.name}</td>
                      <td className="py-3.5">
                        <span
                          className={`px-2.5 py-0.5 rounded-full text-xs font-semibold uppercase ${
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
                      </td>
                      <td className="py-3.5">
                        <div className="flex items-center gap-2 max-w-xs">
                          <div className="w-full bg-slate-850 h-1.5 rounded-full overflow-hidden">
                            <div className="bg-blue-500 h-1.5 rounded-full" style={{ width: `${percent}%` }}></div>
                          </div>
                          <span className="font-mono text-xs text-slate-400">{percent}%</span>
                        </div>
                      </td>
                      <td className="py-3.5 font-mono text-slate-300">{job.agents_saved}</td>
                      <td className="py-3.5 text-slate-400 text-xs">
                        {job.started_at ? new Date(job.started_at).toLocaleString() : "Pending"}
                      </td>
                      <td className="py-3.5 text-right">
                        <Link
                          href={`/jobs/${job.id}`}
                          className="bg-slate-900 text-slate-200 border border-slate-800 hover:bg-slate-850 px-3 py-1 rounded-md text-xs font-medium inline-block transition-colors"
                        >
                          View Details
                        </Link>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
