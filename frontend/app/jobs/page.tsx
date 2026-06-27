"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

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
	created_at: string;
}

export default function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  const fetchJobs = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/jobs`);
      if (!res.ok) throw new Error("Failed to fetch jobs");
      const data = await res.json();
      setJobs(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchJobs();

    // Poll every 3 seconds to update progress bars/status in real-time
    const interval = setInterval(fetchJobs, 3000);
    return () => clearInterval(interval);
  }, [API_BASE]);

  const handleAction = async (jobID: string, action: string) => {
    try {
      const res = await fetch(`${API_BASE}/api/jobs/${jobID}/${action}`, {
        method: "POST",
      });
      if (!res.ok) throw new Error(`Failed to ${action} job`);
      fetchJobs();
    } catch (err: any) {
      alert(`Action error: ${err.message}`);
    }
  };

  if (loading && jobs.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-12">
        <div className="text-slate-400 flex flex-col items-center gap-3">
          <div className="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
          <p className="font-medium">Loading jobs database...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8 max-w-7xl mx-auto w-full space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Scrape Jobs</h2>
          <p className="text-slate-400 mt-1">Manage and track background crawler runs.</p>
        </div>
        <Link
          href="/jobs/new"
          className="bg-blue-600 hover:bg-blue-500 text-white font-semibold text-sm px-5 py-2.5 rounded-lg shadow-lg shadow-blue-950/20 transition-all flex items-center gap-2"
        >
          ➕ Start New Scrape
        </Link>
      </div>

      {error && jobs.length === 0 && (
        <div className="bg-red-950/40 border border-red-800 text-red-200 p-4 rounded-lg text-sm">
          ⚠️ Connection Failure: {error}
        </div>
      )}

      {jobs.length === 0 ? (
        <div className="text-center py-16 text-slate-500 border-2 border-dashed border-slate-800 rounded-xl space-y-4">
          <span className="text-4xl block">🕷️</span>
          <p className="text-base font-medium">No scrape runs registered in the database.</p>
          <Link
            href="/jobs/new"
            className="inline-block bg-slate-900 border border-slate-800 text-slate-300 hover:bg-slate-850 px-4 py-2 rounded-lg text-xs font-semibold"
          >
            Create Your First Scrape Job
          </Link>
        </div>
      ) : (
        <div className="bg-slate-950 rounded-xl border border-slate-800 p-6">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-sm">
              <thead>
                <tr className="border-b border-slate-800 text-slate-400">
                  <th className="pb-3.5 font-semibold">Job Name</th>
                  <th className="pb-3.5 font-semibold">Status</th>
                  <th className="pb-3.5 font-semibold">Request Progress</th>
                  <th className="pb-3.5 font-semibold">Agents Saved</th>
                  <th className="pb-3.5 font-semibold">Stats (OK / Fail)</th>
                  <th className="pb-3.5 font-semibold">Start Time</th>
                  <th className="pb-3.5 font-semibold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-900">
                {jobs.map((job) => {
                  const percent = job.total_estimated_requests > 0
                    ? Math.round((job.completed_requests / job.total_estimated_requests) * 100)
                    : 0;

                  return (
                    <tr key={job.id} className="hover:bg-slate-900/40">
                      <td className="py-4 font-semibold text-slate-200">
                        <Link href={`/jobs/${job.id}`} className="hover:text-blue-400 transition-colors">
                          {job.name}
                        </Link>
                      </td>
                      <td className="py-4">
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
                      <td className="py-4">
                        <div className="flex items-center gap-2.5 max-w-xs">
                          <div className="w-full bg-slate-850 h-1.5 rounded-full overflow-hidden">
                            <div className="bg-blue-500 h-1.5 rounded-full" style={{ width: `${percent}%` }}></div>
                          </div>
                          <span className="font-mono text-xs text-slate-400">{percent}%</span>
                        </div>
                      </td>
                      <td className="py-4 font-semibold text-slate-200 font-mono">{job.agents_saved}</td>
                      <td className="py-4 font-mono text-xs text-slate-400">
                        <span className="text-green-400">{job.completed_requests}</span> / <span className="text-red-400">{job.failed_requests}</span>
                      </td>
                      <td className="py-4 text-xs text-slate-400 font-mono">
                        {job.started_at ? new Date(job.started_at).toLocaleString() : "Pending"}
                      </td>
                      <td className="py-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Link
                            href={`/jobs/${job.id}`}
                            className="bg-slate-900 border border-slate-800 text-slate-200 hover:bg-slate-850 px-3 py-1.5 rounded-md text-xs font-semibold transition-colors"
                          >
                            Details
                          </Link>

                          {job.status === "running" && (
                            <button
                              onClick={() => handleAction(job.id, "pause")}
                              className="bg-yellow-950/80 text-yellow-300 border border-yellow-850 hover:bg-yellow-900 px-3 py-1.5 rounded-md text-xs font-semibold transition-colors"
                            >
                              Pause
                            </button>
                          )}

                          {job.status === "paused" && (
                            <button
                              onClick={() => handleAction(job.id, "resume")}
                              className="bg-green-950/80 text-green-300 border border-green-850 hover:bg-green-900 px-3 py-1.5 rounded-md text-xs font-semibold transition-colors"
                            >
                              Resume
                            </button>
                          )}

                          {(job.status === "running" || job.status === "paused") && (
                            <button
                              onClick={() => handleAction(job.id, "cancel")}
                              className="bg-red-950/80 text-red-300 border border-red-850 hover:bg-red-900 px-3 py-1.5 rounded-md text-xs font-semibold transition-colors"
                            >
                              Cancel
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
