"use client";

import { useEffect, useState } from "react";

interface Agent {
	id: string;
	name: string;
	title: string;
	slogan: string;
	email: string;
	rating: number;
	profile_url: string;
	photo: string;
	city: string;
	state: string;
	zip: string;
	brokerage: string;
	office_name: string;
	phones: string[];
	languages: string[];
	areas: string[];
}

interface FullAgentDetail extends Agent {
	first_name: string;
	last_name: string;
	description: string;
	recommendations_count: number;
	review_count: number;
	website: string;
	video: string;
	address_line_1: string;
	address_line_2: string;
	office_email: string;
	office_website: string;
	office_slogan: string;
	licenses: string[];
	mls_codes: string[];
	social_profiles: string[];
}

export default function AgentsPage() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit] = useState(25);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Selected Rows for Export
  const [selectedAgentIDs, setSelectedAgentIDs] = useState<string[]>([]);

  // Detailed Modal State
  const [selectedAgentDetail, setSelectedAgentDetail] = useState<FullAgentDetail | null>(null);
  const [modalLoading, setModalLoading] = useState(false);

  // Filters State
  const [search, setSearch] = useState("");
  const [state, setState] = useState("");
  const [city, setCity] = useState("");
  const [zip, setZip] = useState("");
  const [brokerage, setBrokerage] = useState("");
  const [hasPhone, setHasPhone] = useState(false);
  const [hasMobile, setHasMobile] = useState(false);
  const [hasOfficePhone, setHasOfficePhone] = useState(false);
  const [language, setLanguage] = useState("");
  const [areaServed, setAreaServed] = useState("");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  const fetchAgents = async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString(),
      });

      if (state) params.append("state", state);
      if (city) params.append("city", city);
      if (zip) params.append("zip", zip);
      if (brokerage) params.append("brokerage", brokerage);
      if (hasPhone) params.append("has_phone", "true");
      if (hasMobile) params.append("has_mobile", "true");
      if (hasOfficePhone) params.append("has_office_phone", "true");
      if (language) params.append("language", language);
      if (areaServed) params.append("area_served", areaServed);

      // Search filters name in backend by matching agent_name
      if (search) params.append("brokerage", search); // falls back or is customized

      const res = await fetch(`${API_BASE}/api/agents?${params.toString()}`);
      if (!res.ok) throw new Error("Failed to fetch agents");
      const data = await res.json();
      setAgents(data.agents || []);
      setTotal(data.total || 0);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAgents();
  }, [page, state, hasPhone, hasMobile, hasOfficePhone, API_BASE]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    fetchAgents();
  };

  const handleResetFilters = () => {
    setSearch("");
    setState("");
    setCity("");
    setZip("");
    setBrokerage("");
    setHasPhone(false);
    setHasMobile(false);
    setHasOfficePhone(false);
    setLanguage(false as any);
    setLanguage("");
    setAreaServed("");
    setPage(1);
  };

  const handleRowSelect = (id: string) => {
    if (selectedAgentIDs.includes(id)) {
      setSelectedAgentIDs(selectedAgentIDs.filter((x) => x !== id));
    } else {
      setSelectedAgentIDs([...selectedAgentIDs, id]);
    }
  };

  const handleSelectAll = () => {
    if (selectedAgentIDs.length === agents.length) {
      setSelectedAgentIDs([]);
    } else {
      setSelectedAgentIDs(agents.map((a) => a.id));
    }
  };

  const viewAgentDetails = async (id: string) => {
    setModalLoading(true);
    setSelectedAgentDetail(null);
    try {
      const res = await fetch(`${API_BASE}/api/agents/${id}`);
      if (!res.ok) throw new Error("Failed to fetch agent profile");
      const data = await res.json();
      setSelectedAgentDetail(data);
    } catch (err: any) {
      alert(err.message);
    } finally {
      setModalLoading(false);
    }
  };

  const handleBulkExportCSV = () => {
    // Generates client-side CSV for only selected items
    if (selectedAgentIDs.length === 0) return;
    const selectedAgents = agents.filter((a) => selectedAgentIDs.includes(a.id));

    let csvContent = "data:text/csv;charset=utf-8,";
    csvContent += "Agent ID,Name,Title,Email,Brokerage,Office,City,State,Zip,Phones,Languages,Areas Served\n";

    selectedAgents.forEach((a) => {
      const row = [
        a.id,
        `"${a.name.replace(/"/g, '""')}"`,
        `"${(a.title || "").replace(/"/g, '""')}"`,
        a.email,
        `"${(a.brokerage || "").replace(/"/g, '""')}"`,
        `"${(a.office_name || "").replace(/"/g, '""')}"`,
        a.city,
        a.state,
        a.zip,
        `"${a.phones.join(", ")}"`,
        `"${a.languages.join(", ")}"`,
        `"${a.areas.join(", ")}"`,
      ].join(",");
      csvContent += row + "\n";
    });

    const encodedUri = encodeURI(csvContent);
    const link = document.createElement("a");
    link.setAttribute("href", encodedUri);
    link.setAttribute("download", "selected_agents.csv");
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const totalPages = Math.ceil(total / limit);

  return (
    <div className="p-8 max-w-7xl mx-auto w-full space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-800 pb-6">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Agent Database</h2>
          <p className="text-slate-400 mt-1">Search, filter, and inspect collected real estate agent profiles.</p>
        </div>
        <div className="flex items-center gap-2">
          {selectedAgentIDs.length > 0 && (
            <button
              onClick={handleBulkExportCSV}
              className="bg-teal-600 hover:bg-teal-500 text-white font-semibold text-sm px-4 py-2.5 rounded-lg transition-colors flex items-center gap-2"
            >
              📥 Export Selected ({selectedAgentIDs.length})
            </button>
          )}
          <button
            onClick={handleResetFilters}
            className="border border-slate-800 hover:bg-slate-850 text-slate-300 px-4 py-2.5 rounded-lg text-sm font-semibold transition-colors"
          >
            Clear Filters
          </button>
        </div>
      </div>

      {/* Main Panel Grid (Filters sidebar + Table) */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
        {/* Left Side: Filter Panels */}
        <div className="lg:col-span-1 bg-slate-950 rounded-xl border border-slate-800 p-5 h-fit space-y-6">
          <div>
            <h3 className="font-semibold text-sm border-b border-slate-900 pb-2">🔍 Search & Filter</h3>
          </div>

          <form onSubmit={handleSearchSubmit} className="space-y-4">
            {/* Keyword Search */}
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Brokerage Search</label>
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="e.g. Century 21"
                className="bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-600 focus:outline-none focus:border-blue-500"
              />
            </div>

            {/* City */}
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider">City</label>
              <input
                type="text"
                value={city}
                onChange={(e) => setCity(e.target.value)}
                placeholder="e.g. San Francisco"
                className="bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-655 focus:outline-none focus:border-blue-500"
              />
            </div>

            {/* Zip */}
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Zip Code</label>
              <input
                type="text"
                value={zip}
                onChange={(e) => setZip(e.target.value)}
                placeholder="e.g. 94102"
                className="bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-655 focus:outline-none focus:border-blue-500"
              />
            </div>

            {/* State dropdown */}
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider">State Code</label>
              <input
                type="text"
                value={state}
                onChange={(e) => setState(e.target.value)}
                placeholder="e.g. CA"
                maxLength={2}
                className="bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 uppercase font-mono placeholder-slate-655 focus:outline-none focus:border-blue-500"
              />
            </div>

            {/* Language */}
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Language</label>
              <input
                type="text"
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                placeholder="e.g. Spanish"
                className="bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-655 focus:outline-none focus:border-blue-500"
              />
            </div>

            {/* Area Served */}
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Area Served</label>
              <input
                type="text"
                value={areaServed}
                onChange={(e) => setAreaServed(e.target.value)}
                placeholder="e.g. Orange County"
                className="bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-655 focus:outline-none focus:border-blue-500"
              />
            </div>

            <button
              type="submit"
              className="w-full bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs py-2 rounded-lg transition-colors"
            >
              Apply Filter Params
            </button>
          </form>

          {/* Checklist Panel */}
          <div className="border-t border-slate-900 pt-4 space-y-3.5">
            <h4 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Contact Checkbox</h4>

            <div className="space-y-2">
              <label className="flex items-center gap-2.5 text-xs text-slate-300 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={hasPhone}
                  onChange={(e) => setHasPhone(e.target.checked)}
                  className="rounded bg-slate-900 border-slate-800 text-blue-500 w-4 h-4 focus:ring-0 focus:ring-offset-0"
                />
                Has Any Phone Number
              </label>

              <label className="flex items-center gap-2.5 text-xs text-slate-300 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={hasMobile}
                  onChange={(e) => setHasMobile(e.target.checked)}
                  className="rounded bg-slate-900 border-slate-800 text-blue-500 w-4 h-4 focus:ring-0 focus:ring-offset-0"
                />
                Has Mobile Number
              </label>

              <label className="flex items-center gap-2.5 text-xs text-slate-300 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={hasOfficePhone}
                  onChange={(e) => setHasOfficePhone(e.target.checked)}
                  className="rounded bg-slate-900 border-slate-800 text-blue-500 w-4 h-4 focus:ring-0 focus:ring-offset-0"
                />
                Has Office Number
              </label>
            </div>
          </div>
        </div>

        {/* Right Side: Agents Table */}
        <div className="lg:col-span-3 bg-slate-950 rounded-xl border border-slate-800 p-6 flex flex-col justify-between min-h-[500px]">
          {loading ? (
            <div className="flex-1 flex items-center justify-center">
              <div className="text-slate-400 flex flex-col items-center gap-3">
                <div className="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
                <p className="text-xs font-semibold">Updating agent tables...</p>
              </div>
            </div>
          ) : agents.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center text-slate-500 gap-2">
              <span className="text-3xl">🫙</span>
              <p className="font-medium text-sm">No agent records found matching your filters.</p>
              <button
                onClick={handleResetFilters}
                className="text-xs text-blue-400 hover:underline font-semibold"
              >
                Reset all search parameters
              </button>
            </div>
          ) : (
            <div className="space-y-6 flex-1 flex flex-col justify-between">
              {/* Actual Table */}
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-400 uppercase tracking-wider font-semibold">
                      <th className="pb-3 w-8">
                        <input
                          type="checkbox"
                          checked={selectedAgentIDs.length === agents.length}
                          onChange={handleSelectAll}
                          className="rounded bg-slate-900 border-slate-800 text-blue-500 w-4 h-4 focus:ring-0 focus:ring-offset-0"
                        />
                      </th>
                      <th className="pb-3">Agent Name</th>
                      <th className="pb-3">Brokerage / Firm</th>
                      <th className="pb-3">Office Name</th>
                      <th className="pb-3">Location (City/State)</th>
                      <th className="pb-3">Phones</th>
                      <th className="pb-3 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-900/60">
                    {agents.map((agent) => (
                      <tr key={agent.id} className="hover:bg-slate-900/20">
                        <td className="py-3">
                          <input
                            type="checkbox"
                            checked={selectedAgentIDs.includes(agent.id)}
                            onChange={() => handleRowSelect(agent.id)}
                            className="rounded bg-slate-900 border-slate-800 text-blue-500 w-4 h-4 focus:ring-0 focus:ring-offset-0"
                          />
                        </td>
                        <td className="py-3">
                          <button
                            onClick={() => viewAgentDetails(agent.id)}
                            className="font-bold text-slate-200 hover:text-blue-400 hover:underline transition-colors text-left"
                          >
                            {agent.name}
                          </button>
                          {agent.title && <span className="block text-[10px] text-slate-500 mt-0.5">{agent.title}</span>}
                        </td>
                        <td className="py-3 text-slate-300 font-medium">{agent.brokerage || "Independent"}</td>
                        <td className="py-3 text-slate-400">{agent.office_name || "N/A"}</td>
                        <td className="py-3 text-slate-400 font-mono">
                          {agent.city ? `${agent.city}, ${agent.state}` : "Unknown"}
                        </td>
                        <td className="py-3">
                          {agent.phones && agent.phones.length > 0 ? (
                            <div className="flex flex-col gap-0.5 max-w-[160px] truncate">
                              {agent.phones.slice(0, 2).map((p, idx) => (
                                <span key={idx} className="font-mono text-slate-300 block">
                                  📞 {p.split(" ")[0]}
                                </span>
                              ))}
                              {agent.phones.length > 2 && (
                                <span className="text-[10px] text-slate-500">+{agent.phones.length - 2} more</span>
                              )}
                            </div>
                          ) : (
                            <span className="text-slate-600 italic">None</span>
                          )}
                        </td>
                        <td className="py-3 text-right">
                          <button
                            onClick={() => viewAgentDetails(agent.id)}
                            className="bg-slate-900 hover:bg-slate-850 text-slate-300 border border-slate-800 px-2.5 py-1.5 rounded-md text-[10px] font-semibold transition-colors"
                          >
                            Profile Details
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination Footer */}
              <div className="flex items-center justify-between border-t border-slate-900 pt-4 text-xs">
                <span className="text-slate-450">
                  Showing <span className="text-slate-200 font-semibold">{(page - 1) * limit + 1}</span> to{" "}
                  <span className="text-slate-200 font-semibold">
                    {Math.min(page * limit, total)}
                  </span>{" "}
                  of <span className="text-slate-200 font-semibold">{total.toLocaleString()}</span> agents
                </span>

                <div className="flex items-center gap-1">
                  <button
                    onClick={() => setPage(Math.max(1, page - 1))}
                    disabled={page === 1}
                    className="border border-slate-800 hover:bg-slate-850 disabled:bg-slate-900/40 disabled:text-slate-600 px-3 py-1.5 rounded text-[11px] font-semibold transition-all"
                  >
                    Previous
                  </button>
                  <span className="px-3 py-1 bg-slate-900 rounded font-mono font-semibold text-slate-300">
                    Page {page} / {totalPages || 1}
                  </span>
                  <button
                    onClick={() => setPage(Math.min(totalPages, page + 1))}
                    disabled={page === totalPages || totalPages === 0}
                    className="border border-slate-800 hover:bg-slate-850 disabled:bg-slate-900/40 disabled:text-slate-600 px-3 py-1.5 rounded text-[11px] font-semibold transition-all"
                  >
                    Next
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Drawer Overlay Modal */}
      {selectedAgentDetail && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex justify-end">
          <div className="w-full max-w-2xl bg-slate-950 border-l border-slate-800 h-full overflow-y-auto p-8 shadow-2xl flex flex-col justify-between animate-slide-in">
            {/* Modal Header */}
            <div className="space-y-6">
              <div className="flex items-start justify-between">
                <div className="flex gap-4 items-center">
                  {selectedAgentDetail.photo ? (
                    <img
                      src={selectedAgentDetail.photo}
                      alt={selectedAgentDetail.name}
                      className="w-16 h-16 rounded-full object-cover border-2 border-slate-800"
                    />
                  ) : (
                    <div className="w-16 h-16 bg-slate-900 border border-slate-850 rounded-full flex items-center justify-center text-2xl">
                      👤
                    </div>
                  )}
                  <div>
                    <h3 className="text-xl font-bold text-slate-100">{selectedAgentDetail.name}</h3>
                    <p className="text-xs text-slate-500 font-semibold">{selectedAgentDetail.title || "Sales Agent"}</p>
                    <p className="text-xs text-blue-400 font-mono font-medium mt-1">{selectedAgentDetail.brokerage}</p>
                  </div>
                </div>
                <button
                  onClick={() => setSelectedAgentDetail(null)}
                  className="text-slate-500 hover:text-slate-300 text-xl font-bold p-2"
                >
                  ✕
                </button>
              </div>

              {selectedAgentDetail.slogan && (
                <p className="text-slate-400 italic text-sm border-l-2 border-blue-500 pl-4 py-1">
                  "{selectedAgentDetail.slogan}"
                </p>
              )}

              {/* Grid Stats */}
              <div className="grid grid-cols-3 gap-4 border-y border-slate-900 py-4 text-xs font-mono text-center">
                <div>
                  <span className="text-slate-500 block">RATING</span>
                  <span className="font-bold text-slate-300 text-sm mt-0.5 block">{selectedAgentDetail.rating || 0} / 5</span>
                </div>
                <div>
                  <span className="text-slate-500 block">RECOMMS</span>
                  <span className="font-bold text-slate-300 text-sm mt-0.5 block">{selectedAgentDetail.recommendations_count || 0}</span>
                </div>
                <div>
                  <span className="text-slate-500 block">REVIEWS</span>
                  <span className="font-bold text-slate-300 text-sm mt-0.5 block">{selectedAgentDetail.review_count || 0}</span>
                </div>
              </div>

              {/* Bio Description */}
              {selectedAgentDetail.description && (
                <div className="space-y-2">
                  <h4 className="text-xs font-bold text-slate-400 uppercase tracking-wider">Biography</h4>
                  <p className="text-slate-300 text-xs leading-relaxed max-h-[160px] overflow-y-auto break-words bg-slate-900/30 p-3.5 rounded-lg border border-slate-900">
                    {selectedAgentDetail.description}
                  </p>
                </div>
              )}

              {/* Contact Information */}
              <div className="space-y-4">
                <h4 className="text-xs font-bold text-slate-400 uppercase tracking-wider">Contact Info</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                  <div className="space-y-2.5">
                    <div>
                      <span className="text-slate-500 block">Email Address</span>
                      <span className="text-slate-300 font-semibold">{selectedAgentDetail.email || "N/A"}</span>
                    </div>
                    <div>
                      <span className="text-slate-500 block">Office Address</span>
                      <span className="text-slate-300 font-semibold leading-relaxed">
                        {selectedAgentDetail.address_line_1} {selectedAgentDetail.address_line_2}
                        <br />
                        {selectedAgentDetail.city}, {selectedAgentDetail.state} {selectedAgentDetail.zip}
                      </span>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <span className="text-slate-500 block">Phones</span>
                    {selectedAgentDetail.phones && selectedAgentDetail.phones.length > 0 ? (
                      <div className="space-y-1">
                        {selectedAgentDetail.phones.map((ph, idx) => (
                          <span key={idx} className="font-mono text-slate-300 block">
                            📞 {ph}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="text-slate-650 italic">None</span>
                    )}
                  </div>
                </div>
              </div>

              {/* Office Details */}
              <div className="space-y-4 border-t border-slate-900 pt-4">
                <h4 className="text-xs font-bold text-slate-400 uppercase tracking-wider">Brokerage Office Details</h4>
                <div className="grid grid-cols-2 gap-4 text-xs">
                  <div>
                    <span className="text-slate-500 block">Office Name</span>
                    <span className="text-slate-300 font-semibold">{selectedAgentDetail.office_name || "N/A"}</span>
                  </div>
                  <div>
                    <span className="text-slate-500 block">Office Email</span>
                    <span className="text-slate-300 font-semibold">{selectedAgentDetail.office_email || "N/A"}</span>
                  </div>
                </div>
              </div>

              {/* Credentials (Languages, Licenses, MLS) */}
              <div className="space-y-4 border-t border-slate-900 pt-4 text-xs">
                <h4 className="text-xs font-bold text-slate-400 uppercase tracking-wider">Licenses & Memberships</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <div>
                      <span className="text-slate-500 block">Languages Spoken</span>
                      <span className="text-slate-300">
                        {selectedAgentDetail.languages && selectedAgentDetail.languages.length > 0
                          ? selectedAgentDetail.languages.join(", ")
                          : "English"}
                      </span>
                    </div>
                    <div>
                      <span className="text-slate-500 block">Evolutionary Licenses</span>
                      <span className="text-slate-300 leading-relaxed block font-mono">
                        {selectedAgentDetail.licenses && selectedAgentDetail.licenses.length > 0
                          ? selectedAgentDetail.licenses.join(", ")
                          : "None"}
                      </span>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <div>
                      <span className="text-slate-500 block">MLS Boards</span>
                      <span className="text-slate-300 block font-mono">
                        {selectedAgentDetail.mls_codes && selectedAgentDetail.mls_codes.length > 0
                          ? selectedAgentDetail.mls_codes.join(", ")
                          : "None"}
                      </span>
                    </div>
                    <div>
                      <span className="text-slate-500 block">Areas Served</span>
                      <span className="text-slate-300 leading-relaxed block">
                        {selectedAgentDetail.areas && selectedAgentDetail.areas.length > 0
                          ? selectedAgentDetail.areas.join(", ")
                          : "N/A"}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Footer Buttons */}
            <div className="mt-8 border-t border-slate-800 pt-6 flex justify-end gap-3">
              {selectedAgentDetail.profile_url && (
                <a
                  href={selectedAgentDetail.profile_url}
                  target="_blank"
                  rel="noreferrer"
                  className="bg-slate-900 border border-slate-800 hover:bg-slate-850 text-slate-200 px-5 py-2.5 rounded-lg text-xs font-semibold transition-all"
                >
                  🔗 View Realtor.com Profile
                </a>
              )}
              <button
                onClick={() => setSelectedAgentDetail(null)}
                className="bg-blue-600 hover:bg-blue-500 text-white px-6 py-2.5 rounded-lg text-xs font-semibold transition-all"
              >
                Close Profile
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
