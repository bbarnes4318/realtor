"use client";

import { useEffect, useState } from "react";

interface ProxyItem {
  id: string;
  url: string;
  status: string;
  created_at: string;
}

interface BalanceResponse {
  wallet_balance: number;
  gb_credits: number;
  credits?: number;
}

interface FlamePackage {
  id: any;
  name: string;
  product: string;
  traffic_max: number;
  traffic_used: number;
  status: string;
  username: string;
  password: string;
}

export default function ProxiesPage() {
  const [proxies, setProxies] = useState<ProxyItem[]>([]);
  const [balance, setBalance] = useState<BalanceResponse | null>(null);
  const [packages, setPackages] = useState<FlamePackage[]>([]);
  const [loadingProxies, setLoadingProxies] = useState(true);
  const [loadingFlame, setLoadingFlame] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Forms
  const [urlInput, setUrlInput] = useState("");
  const [bulkInput, setBulkInput] = useState("");
  const [gbAmount, setGbAmount] = useState(5);
  const [selectedProduct, setSelectedProduct] = useState("residential");
  const [isAdding, setIsAdding] = useState(false);
  const [isOrdering, setIsOrdering] = useState(false);
  const [statusMessage, setStatusMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [activeTab, setActiveTab] = useState<"pool" | "flame">("pool");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  const fetchProxies = async () => {
    try {
      setLoadingProxies(true);
      const res = await fetch(`${API_BASE}/api/proxies`);
      if (!res.ok) throw new Error("Failed to fetch proxies pool");
      const data = await res.json();
      setProxies(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error(err);
      setError(err.message || "Failed to load proxies");
    } finally {
      setLoadingProxies(false);
    }
  };

  const fetchFlameData = async () => {
    try {
      setLoadingFlame(true);
      // Fetch balance
      const balanceRes = await fetch(`${API_BASE}/api/flame-proxies/balance`);
      if (balanceRes.ok) {
        const balanceData = await balanceRes.json();
        setBalance(balanceData);
      }
      
      // Fetch packages
      const packagesRes = await fetch(`${API_BASE}/api/flame-proxies/packages`);
      if (packagesRes.ok) {
        const packagesData = await packagesRes.json();
        setPackages(Array.isArray(packagesData) ? packagesData : []);
      }
    } catch (err) {
      console.error("Failed to load Flame Proxies data", err);
    } finally {
      setLoadingFlame(false);
    }
  };

  useEffect(() => {
    fetchProxies();
    fetchFlameData();
  }, []);

  const handleAddProxy = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!urlInput.trim()) return;
    try {
      setIsAdding(true);
      setStatusMessage(null);
      const res = await fetch(`${API_BASE}/api/proxies`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: urlInput.trim() }),
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || "Failed to add proxy");
      }
      setUrlInput("");
      setStatusMessage({ type: "success", text: "Proxy added successfully!" });
      fetchProxies();
    } catch (err: any) {
      setStatusMessage({ type: "error", text: err.message });
    } finally {
      setIsAdding(false);
    }
  };

  const handleAddBulk = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!bulkInput.trim()) return;
    try {
      setIsAdding(true);
      setStatusMessage(null);
      const res = await fetch(`${API_BASE}/api/proxies`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ bulk: bulkInput.trim() }),
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || "Failed to import proxies");
      }
      const result = await res.json();
      setBulkInput("");
      setStatusMessage({
        type: "success",
        text: `Successfully imported ${result.imported} proxies. Errors: ${result.errors?.length || 0}`,
      });
      fetchProxies();
    } catch (err: any) {
      setStatusMessage({ type: "error", text: err.message });
    } finally {
      setIsAdding(false);
    }
  };

  const handleDeleteProxy = async (id: string) => {
    if (!confirm("Are you sure you want to remove this proxy from the rotation pool?")) return;
    try {
      const res = await fetch(`${API_BASE}/api/proxies/${id}`, {
        method: "DELETE",
      });
      if (!res.ok) throw new Error("Failed to delete proxy");
      fetchProxies();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleToggleStatus = async (id: string, currentStatus: string) => {
    const nextStatus = currentStatus === "active" ? "failed" : "active";
    try {
      const res = await fetch(`${API_BASE}/api/proxies/${id}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: nextStatus }),
      });
      if (!res.ok) throw new Error("Failed to update status");
      fetchProxies();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleOrderPackage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (gbAmount <= 0) return;
    try {
      setIsOrdering(true);
      setStatusMessage(null);
      const res = await fetch(`${API_BASE}/api/flame-proxies/orders`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ product: selectedProduct, gb_amount: gbAmount }),
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || "Failed to order package");
      }
      setStatusMessage({ type: "success", text: "Successfully purchased Flame package!" });
      fetchFlameData();
    } catch (err: any) {
      setStatusMessage({ type: "error", text: err.message });
    } finally {
      setIsOrdering(false);
    }
  };

  const handleAddData = async (pkgId: any, amount: number) => {
    const confirmTopUp = confirm(`Top up this package by ${amount} GB? This will consume your Flame balance.`);
    if (!confirmTopUp) return;
    try {
      const res = await fetch(`${API_BASE}/api/flame-proxies/packages/${pkgId}/add-data`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gb_amount: amount }),
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || "Failed to top up data");
      }
      alert("Traffic added successfully!");
      fetchFlameData();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const activeCount = proxies.filter((p) => p.status === "active").length;
  const failedCount = proxies.filter((p) => p.status === "failed").length;

  return (
    <div className="p-8 max-w-7xl mx-auto w-full space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-800 pb-6">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight bg-gradient-to-r from-blue-400 via-indigo-400 to-purple-400 bg-clip-text text-transparent">
            Proxy Rotation Control Panel
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Configure dynamic proxy rotators for outbound scraper requests and manage Flame Proxies packages.
          </p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={() => {
              fetchProxies();
              fetchFlameData();
            }}
            className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-lg transition text-sm font-medium border border-slate-700 flex items-center gap-2"
          >
            <span>🔄</span> Force Refresh
          </button>
        </div>
      </div>

      {/* Stats Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden group">
          <div className="absolute top-0 right-0 w-24 h-24 bg-blue-500/10 rounded-full blur-2xl group-hover:bg-blue-500/20 transition-all duration-500"></div>
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Total Configured</p>
          <p className="text-3xl font-bold text-white mt-2 font-mono">{proxies.length}</p>
          <div className="text-xs text-slate-500 mt-1">Gateways in sqlite pool</div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden group">
          <div className="absolute top-0 right-0 w-24 h-24 bg-emerald-500/10 rounded-full blur-2xl group-hover:bg-emerald-500/20 transition-all duration-500"></div>
          <p className="text-xs font-semibold text-emerald-400 uppercase tracking-wider">Active Pool</p>
          <p className="text-3xl font-bold text-emerald-400 mt-2 font-mono">{activeCount}</p>
          <div className="text-xs text-slate-500 mt-1">Ready for round-robin</div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden group">
          <div className="absolute top-0 right-0 w-24 h-24 bg-red-500/10 rounded-full blur-2xl group-hover:bg-red-500/20 transition-all duration-500"></div>
          <p className="text-xs font-semibold text-red-400 uppercase tracking-wider">Failed Pool</p>
          <p className="text-3xl font-bold text-red-400 mt-2 font-mono">{failedCount}</p>
          <div className="text-xs text-slate-500 mt-1">Temporarily offline</div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg relative overflow-hidden group">
          <div className="absolute top-0 right-0 w-24 h-24 bg-indigo-500/10 rounded-full blur-2xl group-hover:bg-indigo-500/20 transition-all duration-500"></div>
          <p className="text-xs font-semibold text-indigo-400 uppercase tracking-wider">Flame Balance</p>
          <p className="text-3xl font-bold text-indigo-400 mt-2 font-mono">
            {balance ? `$${balance.wallet_balance.toFixed(2)}` : "—"}
          </p>
          <div className="text-xs text-slate-500 mt-1">
            {balance ? `${balance.gb_credits.toFixed(1)} GB Credits Available` : "Fetch error / Not config"}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-800 flex gap-6">
        <button
          onClick={() => setActiveTab("pool")}
          className={`pb-4 text-sm font-semibold border-b-2 transition-all ${
            activeTab === "pool"
              ? "border-blue-500 text-blue-400"
              : "border-transparent text-slate-400 hover:text-white"
          }`}
        >
          🌐 Proxy Rotation Pool ({proxies.length})
        </button>
        <button
          onClick={() => setActiveTab("flame")}
          className={`pb-4 text-sm font-semibold border-b-2 transition-all ${
            activeTab === "flame"
              ? "border-indigo-500 text-indigo-400"
              : "border-transparent text-slate-400 hover:text-white"
          }`}
        >
          🔥 Flame Proxies API Control
        </button>
      </div>

      {/* Status Messages */}
      {statusMessage && (
        <div
          className={`p-4 rounded-lg border text-sm flex justify-between items-center ${
            statusMessage.type === "success"
              ? "bg-emerald-950/40 border-emerald-900 text-emerald-300"
              : "bg-red-950/40 border-red-900 text-red-300"
          }`}
        >
          <span>{statusMessage.text}</span>
          <button onClick={() => setStatusMessage(null)} className="text-xs text-slate-400 hover:text-white">
            ✕
          </button>
        </div>
      )}

      {error && (
        <div className="p-4 bg-red-950/40 border border-red-900 text-red-300 rounded-lg text-sm">
          ⚠️ {error}
        </div>
      )}

      {/* TAB CONTENT: Proxy Rotation Pool */}
      {activeTab === "pool" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Main Proxy Table (Left side 2 cols) */}
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-slate-950 border border-slate-800 rounded-xl overflow-hidden shadow-2xl">
              <div className="p-5 border-b border-slate-800 bg-slate-900/50 flex justify-between items-center">
                <h3 className="font-bold text-white">Configured Proxy Endpoints</h3>
                <span className="text-xs text-slate-400">Round-Robin sequence enabled</span>
              </div>

              {loadingProxies ? (
                <div className="p-12 text-center text-slate-500">
                  <div className="animate-spin inline-block w-8 h-8 border-4 border-slate-700 border-t-blue-500 rounded-full mb-3"></div>
                  <p>Loading rotation pool...</p>
                </div>
              ) : proxies.length === 0 ? (
                <div className="p-16 text-center">
                  <div className="text-4xl mb-4">📭</div>
                  <h4 className="text-white font-bold text-lg">No proxies added yet</h4>
                  <p className="text-slate-500 text-sm max-w-sm mx-auto mt-1">
                    Your scrapers are currently routing requests directly from the host IP. Add proxies on the right to start rotation.
                  </p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-900 text-slate-400 uppercase text-xs tracking-wider border-b border-slate-800">
                      <tr>
                        <th className="p-4 font-semibold">Proxy Gateway URL</th>
                        <th className="p-4 font-semibold">Status</th>
                        <th className="p-4 font-semibold">Added</th>
                        <th className="p-4 font-semibold text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800/60">
                      {proxies.map((p) => {
                        // Mask credentials in URL for safety
                        let maskedURL = p.url;
                        try {
                          const parsed = new URL(p.url);
                          if (parsed.username) {
                            maskedURL = `${parsed.protocol}//${parsed.username}:****@${parsed.host}`;
                          }
                        } catch {}

                        return (
                          <tr key={p.id} className="hover:bg-slate-900/30 transition-all">
                            <td className="p-4 font-mono text-xs text-slate-300 max-w-md truncate">
                              {maskedURL}
                            </td>
                            <td className="p-4">
                              <button
                                onClick={() => handleToggleStatus(p.id, p.status)}
                                className={`px-2.5 py-0.5 rounded-full text-xs font-semibold uppercase tracking-wider transition ${
                                  p.status === "active"
                                    ? "bg-emerald-950/60 border border-emerald-800 text-emerald-400 hover:bg-emerald-900"
                                    : "bg-red-950/60 border border-red-800 text-red-400 hover:bg-red-900"
                                }`}
                              >
                                {p.status}
                              </button>
                            </td>
                            <td className="p-4 text-xs text-slate-400">
                              {new Date(p.created_at).toLocaleDateString()}
                            </td>
                            <td className="p-4 text-right">
                              <button
                                onClick={() => handleDeleteProxy(p.id)}
                                className="text-xs text-slate-400 hover:text-red-400 transition bg-slate-900 hover:bg-red-950/50 p-2 rounded-lg border border-slate-800"
                              >
                                🗑️ Delete
                              </button>
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

          {/* Add Proxy sidebar (Right side 1 col) */}
          <div className="space-y-6">
            {/* Single Input Form */}
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg space-y-4">
              <h3 className="font-bold text-white flex items-center gap-2 text-sm">
                <span>➕</span> Add Custom Proxy
              </h3>
              <form onSubmit={handleAddProxy} className="space-y-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">
                    Proxy URL string
                  </label>
                  <input
                    type="text"
                    value={urlInput}
                    onChange={(e) => setUrlInput(e.target.value)}
                    placeholder="http://user:pass@host:port"
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-blue-500 font-mono"
                  />
                </div>
                <button
                  type="submit"
                  disabled={isAdding}
                  className="w-full py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 text-white rounded-lg transition text-xs font-semibold"
                >
                  {isAdding ? "Adding..." : "Add to Rotation Pool"}
                </button>
              </form>
            </div>

            {/* Bulk Text Area Form */}
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg space-y-4">
              <h3 className="font-bold text-white flex items-center gap-2 text-sm">
                <span>📋</span> Bulk Import Gateways
              </h3>
              <p className="text-xs text-slate-400 leading-relaxed">
                Paste proxies list. Supports standard format or Flame Proxies format (<code className="text-blue-400">host:port:user:pass</code>), newline separated.
              </p>
              <form onSubmit={handleAddBulk} className="space-y-3">
                <div>
                  <textarea
                    rows={6}
                    value={bulkInput}
                    onChange={(e) => setBulkInput(e.target.value)}
                    placeholder="proxy.example.com:8989:username:password&#10;http://user2:pass2@proxy2.com:1234"
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-xs text-slate-100 placeholder-slate-600 focus:outline-none focus:border-blue-500 font-mono"
                  />
                </div>
                <button
                  type="submit"
                  disabled={isAdding}
                  className="w-full py-2 bg-slate-800 hover:bg-slate-700 disabled:bg-slate-900 text-white rounded-lg transition text-xs font-semibold border border-slate-700"
                >
                  {isAdding ? "Importing..." : "Process Bulk Import"}
                </button>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* TAB CONTENT: Flame Proxies API Control */}
      {activeTab === "flame" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Flame Packages (Left side 2 cols) */}
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-slate-950 border border-slate-800 rounded-xl overflow-hidden shadow-2xl">
              <div className="p-5 border-b border-slate-800 bg-slate-900/50 flex justify-between items-center">
                <h3 className="font-bold text-white">My Active Flame Packages</h3>
                <span className="text-xs text-slate-400">Residential proxy bundles</span>
              </div>

              {loadingFlame ? (
                <div className="p-12 text-center text-slate-500">
                  <div className="animate-spin inline-block w-8 h-8 border-4 border-slate-700 border-t-indigo-500 rounded-full mb-3"></div>
                  <p>Fetching packages from API...</p>
                </div>
              ) : packages.length === 0 ? (
                <div className="p-16 text-center">
                  <div className="text-4xl mb-4">💳</div>
                  <h4 className="text-white font-bold text-lg">No active proxy packages</h4>
                  <p className="text-slate-500 text-sm max-w-sm mx-auto mt-1">
                    Buy your first package on the right to instantiate high-quality residential scraper nodes.
                  </p>
                </div>
              ) : (
                <div className="divide-y divide-slate-800">
                  {packages.map((pkg) => {
                    const usagePercent =
                      pkg.traffic_max > 0 ? (pkg.traffic_used / pkg.traffic_max) * 100 : 0;

                    return (
                      <div key={pkg.id} className="p-6 hover:bg-slate-900/20 transition space-y-4">
                        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3">
                          <div>
                            <span className="text-xs bg-indigo-950 border border-indigo-800 text-indigo-300 font-semibold px-2 py-0.5 rounded-full uppercase tracking-wider">
                              {pkg.product.replace("_", " ")}
                            </span>
                            <h4 className="text-white font-bold mt-1 text-base">
                              Package ID: {pkg.id} ({pkg.name || "Unnamed"})
                            </h4>
                          </div>
                          <div className="flex items-center gap-2">
                            <span
                              className={`px-2 py-0.5 text-xs rounded-full font-medium capitalize ${
                                pkg.status === "active"
                                  ? "bg-emerald-950/60 text-emerald-400 border border-emerald-800"
                                  : "bg-slate-900 text-slate-400 border border-slate-800"
                              }`}
                            >
                              {pkg.status}
                            </span>
                          </div>
                        </div>

                        {/* Progress bar */}
                        <div>
                          <div className="flex justify-between text-xs text-slate-400 mb-1 font-mono">
                            <span>Traffic Usage: {pkg.traffic_used.toFixed(2)} GB used</span>
                            <span>Max Limit: {pkg.traffic_max.toFixed(1)} GB</span>
                          </div>
                          <div className="w-full bg-slate-850 rounded-full h-3.5 border border-slate-800 overflow-hidden p-0.5">
                            <div
                              className="bg-indigo-500 h-full rounded-full transition-all duration-500"
                              style={{ width: `${Math.min(usagePercent, 100)}%` }}
                            ></div>
                          </div>
                        </div>

                        {/* Credentials / Details */}
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 bg-slate-900/40 p-4 rounded-xl border border-slate-800/80">
                          <div className="space-y-1">
                            <p className="text-[10px] uppercase font-semibold text-slate-500">Gateway Username</p>
                            <p className="text-xs text-slate-200 font-mono truncate">{pkg.username}</p>
                          </div>
                          <div className="space-y-1">
                            <p className="text-[10px] uppercase font-semibold text-slate-500">Gateway Password</p>
                            <p className="text-xs text-slate-200 font-mono truncate">{pkg.password}</p>
                          </div>
                        </div>

                        {/* Actions */}
                        <div className="flex justify-end gap-3 pt-2">
                          <button
                            onClick={() => handleAddData(pkg.id, 5)}
                            className="px-3.5 py-1.5 bg-indigo-650 hover:bg-indigo-600 border border-indigo-500 text-white rounded-lg text-xs font-semibold transition"
                          >
                            ➕ Add 5GB Traffic
                          </button>
                          <button
                            onClick={() => handleAddData(pkg.id, 10)}
                            className="px-3.5 py-1.5 bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-200 rounded-lg text-xs font-semibold transition"
                          >
                            ➕ Add 10GB Traffic
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>

          {/* Buy package sidebar (Right side 1 col) */}
          <div className="space-y-6">
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-lg space-y-4">
              <h3 className="font-bold text-white flex items-center gap-2 text-sm">
                <span>🛒</span> Purchase Proxy Bundle
              </h3>
              <p className="text-xs text-slate-400 leading-relaxed">
                Order additional residential proxy package credits using your wallet balance.
              </p>
              <form onSubmit={handleOrderPackage} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">
                    Select Product Grade
                  </label>
                  <select
                    value={selectedProduct}
                    onChange={(e) => setSelectedProduct(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-sm text-slate-100 focus:outline-none focus:border-indigo-500"
                  >
                    <option value="residential">Standard Residential ($0.49/GB)</option>
                    <option value="premium_residential">Premium Residential ($2.15/GB)</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">
                    Traffic Amount (GB)
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="1000"
                    value={gbAmount}
                    onChange={(e) => setGbAmount(parseInt(e.target.value) || 0)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-sm text-slate-100 focus:outline-none focus:border-indigo-500 font-mono"
                  />
                </div>

                <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800 text-xs text-slate-400 space-y-1.5">
                  <div className="flex justify-between">
                    <span>Est. Unit Price:</span>
                    <span className="font-semibold text-slate-200">
                      {selectedProduct === "residential" ? "$0.49 / GB" : "$2.15 / GB"}
                    </span>
                  </div>
                  <div className="flex justify-between border-t border-slate-800 pt-1.5 font-bold">
                    <span>Total Cost:</span>
                    <span className="text-indigo-400">
                      ${(gbAmount * (selectedProduct === "residential" ? 0.49 : 2.15)).toFixed(2)}
                    </span>
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={isOrdering}
                  className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-850 text-white rounded-lg transition text-xs font-bold shadow-lg shadow-indigo-650/10"
                >
                  {isOrdering ? "Placing Order..." : "Confirm & Order Package"}
                </button>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
