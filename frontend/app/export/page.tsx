"use client";

import { useEffect, useState } from "react";

interface Job {
	id: string;
	name: string;
	status: string;
	agents_saved: number;
}

export default function ExportPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);

  // Export Params
  const [exportType, setExportType] = useState("all"); // all, job, location
  const [selectedJobID, setSelectedJobID] = useState("");
  const [state, setState] = useState("");
  const [city, setCity] = useState("");
  const [zip, setZip] = useState("");
  const [dedupePhone, setDedupePhone] = useState(true);
  const [dedupeURL, setDedupeURL] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  useEffect(() => {
    async function fetchJobs() {
      try {
        const res = await fetch(`${API_BASE}/api/jobs`);
        if (res.ok) {
          const data = await res.json();
          setJobs(data);
          if (data.length > 0) {
            setSelectedJobID(data[0].id);
          }
        }
      } catch (err) {
        console.error("Failed to load jobs list", err);
      } finally {
        setLoading(false);
      }
    }
    fetchJobs();
  }, [API_BASE]);

  const handleDownload = () => {
    let url = "";
    const params = new URLSearchParams();
    if (dedupePhone) params.append("dedupe_phone", "true");
    if (dedupeURL) params.append("dedupe_url", "true");

    if (exportType === "job" && selectedJobID) {
      url = `${API_BASE}/api/export/jobs/${selectedJobID}.csv?${params.toString()}`;
    } else {
      url = `${API_BASE}/api/export/agents.csv?${params.toString()}`;
      if (exportType === "location") {
        if (state) params.append("state", state);
        if (city) params.append("city", city);
        if (zip) params.append("zip", zip);
        url = `${API_BASE}/api/export/agents.csv?${params.toString()}`;
      }
    }

    // Trigger download
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", ""); // backend sets filename in header
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="p-8 max-w-4xl mx-auto w-full space-y-8">
      <div>
        <h2 className="text-3xl font-bold tracking-tight">Export Center</h2>
        <p className="text-slate-400 mt-1">Generate and download CSV datasets with phone and URL deduplication.</p>
      </div>

      <div className="bg-slate-950 border border-slate-800 rounded-xl p-6 space-y-6">
        <h3 className="text-lg font-semibold border-b border-slate-800 pb-3">📦 CSV Generator Configuration</h3>

        {/* Export Scope Select */}
        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium text-slate-300">Data Selection Scope</label>
          <div className="grid grid-cols-3 gap-4">
            <button
              onClick={() => setExportType("all")}
              className={`p-4 rounded-xl border text-sm font-semibold transition-all flex flex-col items-center gap-2 ${
                exportType === "all"
                  ? "bg-blue-950/40 border-blue-500 text-blue-300"
                  : "bg-slate-900 border-slate-800 text-slate-400 hover:bg-slate-850"
              }`}
            >
              <span className="text-lg">🌍</span> Export All Agents
            </button>
            <button
              onClick={() => setExportType("job")}
              className={`p-4 rounded-xl border text-sm font-semibold transition-all flex flex-col items-center gap-2 ${
                exportType === "job"
                  ? "bg-blue-950/40 border-blue-500 text-blue-300"
                  : "bg-slate-900 border-slate-800 text-slate-400 hover:bg-slate-850"
              }`}
            >
              <span className="text-lg">⚙️</span> Export by Scrape Run
            </button>
            <button
              onClick={() => setExportType("location")}
              className={`p-4 rounded-xl border text-sm font-semibold transition-all flex flex-col items-center gap-2 ${
                exportType === "location"
                  ? "bg-blue-950/40 border-blue-500 text-blue-300"
                  : "bg-slate-900 border-slate-800 text-slate-400 hover:bg-slate-850"
              }`}
            >
              <span className="text-lg">📍</span> Export by Location
            </button>
          </div>
        </div>

        {/* Scope Option Inputs */}
        {exportType === "job" && (
          <div className="flex flex-col gap-2 animate-fade-in">
            <label htmlFor="jobID" className="text-sm font-medium text-slate-300">
              Select Scrape Run Target
            </label>
            {loading ? (
              <div className="text-xs text-slate-500">Loading jobs list...</div>
            ) : jobs.length === 0 ? (
              <div className="text-xs text-yellow-500 italic">No runs available to select from.</div>
            ) : (
              <select
                id="jobID"
                value={selectedJobID}
                onChange={(e) => setSelectedJobID(e.target.value)}
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 focus:outline-none focus:border-blue-500 transition-colors"
              >
                {jobs.map((j) => (
                  <option key={j.id} value={j.id}>
                    {j.name} ({j.agents_saved} agents)
                  </option>
                ))}
              </select>
            )}
          </div>
        )}

        {exportType === "location" && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 animate-fade-in">
            {/* State */}
            <div className="flex flex-col gap-2">
              <label htmlFor="state" className="text-sm font-medium text-slate-300">
                State Code (2-Letters)
              </label>
              <input
                id="state"
                type="text"
                value={state}
                onChange={(e) => setState(e.target.value)}
                placeholder="e.g. CA"
                maxLength={2}
                className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-100 uppercase font-mono focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>

            {/* City */}
            <div className="flex flex-col gap-2">
              <label htmlFor="city" className="text-sm font-medium text-slate-300">
                City Name
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

            {/* Zip */}
            <div className="flex flex-col gap-2">
              <label htmlFor="zip" className="text-sm font-medium text-slate-300">
                Zip Code
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
          </div>
        )}

        {/* Deduplication Panel */}
        <div className="border-t border-slate-800 pt-5 space-y-4">
          <h4 className="text-sm font-semibold text-slate-300">⚔️ Row Deduplication Options</h4>

          <div className="flex flex-col md:flex-row gap-6">
            <label className="flex items-center gap-3 bg-slate-900 border border-slate-800 hover:border-slate-750 px-5 py-3 rounded-xl cursor-pointer select-none flex-1">
              <input
                type="checkbox"
                checked={dedupePhone}
                onChange={(e) => setDedupePhone(e.target.checked)}
                className="w-5 h-5 rounded bg-slate-950 border-slate-750 text-blue-500 focus:ring-0 focus:ring-offset-0"
              />
              <div>
                <span className="text-sm font-semibold text-slate-200 block">Deduplicate by Phone Numbers</span>
                <span className="text-xs text-slate-500 mt-0.5 block">Skips agent records containing previously seen phone numbers.</span>
              </div>
            </label>

            <label className="flex items-center gap-3 bg-slate-900 border border-slate-800 hover:border-slate-750 px-5 py-3 rounded-xl cursor-pointer select-none flex-1">
              <input
                type="checkbox"
                checked={dedupeURL}
                onChange={(e) => setDedupeURL(e.target.checked)}
                className="w-5 h-5 rounded bg-slate-950 border-slate-750 text-blue-500 focus:ring-0 focus:ring-offset-0"
              />
              <div>
                <span className="text-sm font-semibold text-slate-200 block">Deduplicate by Profile URL</span>
                <span className="text-xs text-slate-500 mt-0.5 block">Skips agents with matching profile web URLs.</span>
              </div>
            </label>
          </div>
        </div>

        {/* Download Trigger */}
        <div className="border-t border-slate-800 pt-6 flex justify-end">
          <button
            onClick={handleDownload}
            className="bg-blue-600 hover:bg-blue-500 text-white font-semibold text-sm px-8 py-3 rounded-lg shadow-lg shadow-blue-950/20 transition-all flex items-center gap-2"
          >
            📥 Generate & Download CSV File
          </button>
        </div>
      </div>
    </div>
  );
}
