"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function NewJobPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form Fields State
  const [name, setName] = useState("");
  const [maxAgentsLimit, setMaxAgentsLimit] = useState(100);
  const [concurrency, setConcurrency] = useState(3);
  const [throttleLimit, setThrottleLimit] = useState(5);
  const [saveRaw, setSaveRaw] = useState(false);
  const [dbMode, setDbMode] = useState("local");

  // Filters State
  const [state, setState] = useState("");
  const [city, setCity] = useState("");
  const [zip, setZip] = useState("");
  const [brokerage, setBrokerage] = useState("");
  const [agentName, setAgentName] = useState("");
  const [areaServed, setAreaServed] = useState("");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError("Job name is required.");
      return;
    }

    setLoading(true);
    setError(null);

    const payload = {
      name,
      max_agents_limit: Number(maxAgentsLimit),
      concurrency: Number(concurrency),
      throttle_request_limit: Number(throttleLimit),
      save_raw_agents: saveRaw,
      db_mode: dbMode,
      filters: {
        state,
        city,
        zip,
        brokerage,
        agent_name: agentName,
        area_served: areaServed,
      },
    };

    try {
      // 1. Create the job
      const createRes = await fetch(`${API_BASE}/api/jobs`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!createRes.ok) {
        throw new Error("Failed to create scrape job on backend");
      }

      const newJob = await createRes.json();

      // 2. Start the job automatically
      const startRes = await fetch(`${API_BASE}/api/jobs/${newJob.id}/start`, {
        method: "POST",
      });

      if (!startRes.ok) {
        throw new Error(`Job created (ID: ${newJob.id}) but failed to auto-start`);
      }

      // Redirect to the jobs list page
      router.push("/jobs");
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-8 max-w-4xl mx-auto w-full space-y-8">
      <div>
        <h2 className="text-3xl font-bold tracking-tight">New Scrape Job</h2>
        <p className="text-slate-400 mt-1">Configure scrape execution parameters and search scope.</p>
      </div>

      {error && (
        <div className="bg-red-950/40 border border-red-800 text-red-200 p-4 rounded-lg text-sm">
          ⚠️ {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-8">
        {/* Core Configuration Panel */}
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 space-y-6">
          <h3 className="text-lg font-semibold border-b border-slate-800 pb-3">⚙️ Execution Settings</h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Job Name */}
            <div className="flex flex-col gap-2">
              <label htmlFor="name" className="text-sm font-medium text-slate-300">
                Job Name <span className="text-red-500">*</span>
              </label>
              <input
                id="name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. California Scrape - June 2026"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition-colors"
                required
              />
            </div>

            {/* Database Mode */}
            <div className="flex flex-col gap-2">
              <label htmlFor="dbMode" className="text-sm font-medium text-slate-300">
                Database Target
              </label>
              <select
                id="dbMode"
                value={dbMode}
                onChange={(e) => setDbMode(e.target.value)}
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 focus:outline-none focus:border-blue-500 transition-colors"
              >
                <option value="local">Local Database (SQLite)</option>
                <option value="turso">Turso Remote Database</option>
              </select>
            </div>

            {/* Max Agents Limit */}
            <div className="flex flex-col gap-2">
              <label htmlFor="maxLimit" className="text-sm font-medium text-slate-300">
                Max Agents Limit (0 for infinite)
              </label>
              <input
                id="maxLimit"
                type="number"
                value={maxAgentsLimit}
                onChange={(e) => setMaxAgentsLimit(Number(e.target.value))}
                min="0"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 font-mono focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* Concurrency Threads */}
            <div className="flex flex-col gap-2">
              <label htmlFor="concurrency" className="text-sm font-medium text-slate-300">
                Concurrency Threads (Default: 3)
              </label>
              <input
                id="concurrency"
                type="number"
                value={concurrency}
                onChange={(e) => setConcurrency(Number(e.target.value))}
                min="1"
                max="10"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 font-mono focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* Throttle Request Limit */}
            <div className="flex flex-col gap-2">
              <label htmlFor="throttleLimit" className="text-sm font-medium text-slate-300">
                Throttle Request Count (Default: 5)
              </label>
              <input
                id="throttleLimit"
                type="number"
                value={throttleLimit}
                onChange={(e) => setThrottleLimit(Number(e.target.value))}
                min="1"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 font-mono focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* Save Raw Agents Checkbox */}
            <div className="flex items-center gap-3 h-full pt-6">
              <input
                id="saveRaw"
                type="checkbox"
                checked={saveRaw}
                onChange={(e) => setSaveRaw(e.target.checked)}
                className="w-5 h-5 rounded bg-slate-900 border-slate-700 text-blue-500 focus:ring-0 focus:ring-offset-0"
              />
              <label htmlFor="saveRaw" className="text-sm font-medium text-slate-300 cursor-pointer select-none">
                Save Raw Agent JSON Payload
              </label>
            </div>
          </div>
        </div>

        {/* Target Filters Panel */}
        <div className="bg-slate-950 p-6 rounded-xl border border-slate-800 space-y-6">
          <div className="border-b border-slate-800 pb-3 flex flex-col gap-1">
            <h3 className="text-lg font-semibold">🔍 Scrape Targeting (Optional)</h3>
            <p className="text-xs text-slate-500">
              Note: Unused fields default to full scrape. Zip and Agent Name are passed directly to API, others post-filter in DB.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Zip code */}
            <div className="flex flex-col gap-2">
              <label htmlFor="zip" className="text-sm font-medium text-slate-300">
                Target Zip/Postal Code
              </label>
              <input
                id="zip"
                type="text"
                value={zip}
                onChange={(e) => setZip(e.target.value)}
                placeholder="e.g. 90210"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 font-mono focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* Agent Name */}
            <div className="flex flex-col gap-2">
              <label htmlFor="agentName" className="text-sm font-medium text-slate-300">
                Target Agent Name
              </label>
              <input
                id="agentName"
                type="text"
                value={agentName}
                onChange={(e) => setAgentName(e.target.value)}
                placeholder="e.g. John Doe"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* City */}
            <div className="flex flex-col gap-2">
              <label htmlFor="city" className="text-sm font-medium text-slate-300">
                Target City
              </label>
              <input
                id="city"
                type="text"
                value={city}
                onChange={(e) => setCity(e.target.value)}
                placeholder="e.g. Beverly Hills"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* State */}
            <div className="flex flex-col gap-2">
              <label htmlFor="state" className="text-sm font-medium text-slate-300">
                Target State (2-Letter Code)
              </label>
              <input
                id="state"
                type="text"
                value={state}
                onChange={(e) => setState(e.target.value)}
                placeholder="e.g. CA"
                maxLength={2}
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 font-mono uppercase focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* Brokerage */}
            <div className="flex flex-col gap-2">
              <label htmlFor="brokerage" className="text-sm font-medium text-slate-300">
                Brokerage Firm
              </label>
              <input
                id="brokerage"
                type="text"
                value={brokerage}
                onChange={(e) => setBrokerage(e.target.value)}
                placeholder="e.g. RE/MAX"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* Area Served */}
            <div className="flex flex-col gap-2">
              <label htmlFor="areaServed" className="text-sm font-medium text-slate-300">
                Operating Area Served
              </label>
              <input
                id="areaServed"
                type="text"
                value={areaServed}
                onChange={(e) => setAreaServed(e.target.value)}
                placeholder="e.g. Orange County"
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>
          </div>
        </div>

        {/* Action Panel */}
        <div className="flex justify-end gap-4">
          <button
            type="button"
            onClick={() => router.push("/")}
            className="px-6 py-2.5 border border-slate-800 hover:bg-slate-850 rounded-lg text-sm font-semibold transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-8 py-2.5 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 text-white rounded-lg text-sm font-semibold shadow-lg shadow-blue-950/20 transition-all flex items-center gap-2"
          >
            {loading ? (
              <>
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                Creating...
              </>
            ) : (
              "Create & Start Job"
            )}
          </button>
        </div>
      </form>
    </div>
  );
}
