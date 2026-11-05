import React, { useState } from 'react';
import { Search, Filter, ExternalLink, Copy, Check, ShieldCheck, ChevronLeft, ChevronRight, User, Hash } from 'lucide-react';
import { AuditEvent } from '../types';

interface AuditFeedProps {
  events: AuditEvent[];
  total: number;
  limit: number;
  offset: number;
  onPageChange: (newOffset: number) => void;
  onFilterChange: (filters: { tenant_id?: string; action?: string; q?: string }) => void;
  onSelectEvent: (event: AuditEvent) => void;
  isLoading: boolean;
}

export const AuditFeed: React.FC<AuditFeedProps> = ({
  events,
  total,
  limit,
  offset,
  onPageChange,
  onFilterChange,
  onSelectEvent,
  isLoading,
}) => {
  const [tenantInput, setTenantInput] = useState('');
  const [actionInput, setActionInput] = useState('');
  const [queryInput, setQueryInput] = useState('');
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onFilterChange({
      tenant_id: tenantInput.trim() || undefined,
      action: actionInput.trim() || undefined,
      q: queryInput.trim() || undefined,
    });
  };

  const copyToClipboard = (text: string, id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const getActionBadgeColor = (action: string) => {
    const act = action.toLowerCase();
    if (act.includes('create') || act.includes('grant') || act.includes('enable')) {
      return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
    }
    if (act.includes('delete') || act.includes('revoke') || act.includes('disable')) {
      return 'bg-rose-500/10 text-rose-400 border-rose-500/20';
    }
    if (act.includes('export') || act.includes('query') || act.includes('read')) {
      return 'bg-sky-500/10 text-sky-400 border-sky-500/20';
    }
    return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
  };

  const totalPages = Math.ceil(total / limit) || 1;
  const currentPage = Math.floor(offset / limit) + 1;

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
      {/* Search & Filter Header */}
      <form onSubmit={handleSearchSubmit} className="p-4 border-b border-slate-800 bg-slate-900/80">
        <div className="flex flex-col md:flex-row items-center gap-3">
          <div className="relative flex-1 w-full">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
            <input
              type="text"
              placeholder="Search actors, actions, targets, or event IDs..."
              value={queryInput}
              onChange={(e) => setQueryInput(e.target.value)}
              className="w-full pl-9 pr-4 py-2 bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-500/50"
            />
          </div>

          <div className="flex items-center gap-2 w-full md:w-auto">
            <input
              type="text"
              placeholder="Tenant ID (e.g. org_acme)"
              value={tenantInput}
              onChange={(e) => setTenantInput(e.target.value)}
              className="w-full md:w-44 px-3 py-2 bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-500/50"
            />

            <input
              type="text"
              placeholder="Action filter"
              value={actionInput}
              onChange={(e) => setActionInput(e.target.value)}
              className="w-full md:w-36 px-3 py-2 bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-500/50"
            />

            <button
              type="submit"
              className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded-xl text-xs font-semibold shrink-0 transition"
            >
              Filter
            </button>
          </div>
        </div>
      </form>

      {/* Events Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-slate-800 text-[11px] font-semibold text-slate-400 uppercase tracking-wider bg-slate-950/40">
              <th className="py-3 px-4">Event ID / Time</th>
              <th className="py-3 px-4">Tenant</th>
              <th className="py-3 px-4">Actor</th>
              <th className="py-3 px-4">Action</th>
              <th className="py-3 px-4">Target Resource</th>
              <th className="py-3 px-4">Merkle Leaf Hash</th>
              <th className="py-3 px-4 text-right">Proof</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60 text-xs">
            {isLoading ? (
              <tr>
                <td colSpan={7} className="py-12 text-center text-slate-500">
                  <div className="inline-block animate-spin rounded-full h-5 w-5 border-2 border-slate-600 border-t-emerald-400 mb-2" />
                  <div>Loading cryptographic records...</div>
                </td>
              </tr>
            ) : events.length === 0 ? (
              <tr>
                <td colSpan={7} className="py-12 text-center text-slate-500">
                  No audit events found matching the filter criteria.
                </td>
              </tr>
            ) : (
              events.map((evt) => (
                <tr
                  key={evt.id}
                  onClick={() => onSelectEvent(evt)}
                  className="hover:bg-slate-800/40 cursor-pointer transition group"
                >
                  {/* Event ID & Time */}
                  <td className="py-3 px-4">
                    <div className="font-mono text-slate-200 font-medium text-[11px] flex items-center gap-1.5">
                      <span>{evt.id}</span>
                      <button
                        onClick={(e) => copyToClipboard(evt.id, evt.id, e)}
                        className="opacity-0 group-hover:opacity-100 text-slate-400 hover:text-slate-200 transition"
                      >
                        {copiedId === evt.id ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                      </button>
                    </div>
                    <div className="text-[10px] text-slate-500 mt-0.5" title={evt.timestamp}>
                      {new Date(evt.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })} • Block #{evt.block_index}
                    </div>
                  </td>

                  {/* Tenant */}
                  <td className="py-3 px-4">
                    <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-slate-800/80 text-slate-300 border border-slate-700">
                      {evt.tenant_id}
                    </span>
                  </td>

                  {/* Actor */}
                  <td className="py-3 px-4">
                    <div className="font-medium text-slate-200 truncate max-w-[140px]">
                      {evt.actor?.name || evt.actor?.id}
                    </div>
                    <div className="text-[10px] text-slate-500 truncate max-w-[140px]">
                      {evt.actor?.email || evt.actor?.role || 'actor'}
                    </div>
                  </td>

                  {/* Action */}
                  <td className="py-3 px-4">
                    <span className={`inline-block px-2.5 py-1 rounded-md text-[11px] font-mono font-medium border ${getActionBadgeColor(evt.action)}`}>
                      {evt.action}
                    </span>
                  </td>

                  {/* Target Resource */}
                  <td className="py-3 px-4">
                    <div className="text-slate-300 font-medium truncate max-w-[160px]">
                      {evt.target?.name || evt.target?.id}
                    </div>
                    <div className="text-[10px] text-slate-500 font-mono">
                      {evt.target?.type}
                    </div>
                  </td>

                  {/* Leaf Hash */}
                  <td className="py-3 px-4 font-mono text-[11px] text-slate-400">
                    <div className="flex items-center gap-1">
                      <span className="truncate max-w-[110px]" title={evt.leaf_hash}>
                        {evt.leaf_hash ? `${evt.leaf_hash.slice(0, 10)}...` : '-'}
                      </span>
                      {evt.leaf_hash && (
                        <button
                          onClick={(e) => copyToClipboard(evt.leaf_hash, `hash_${evt.id}`, e)}
                          className="opacity-0 group-hover:opacity-100 text-slate-500 hover:text-slate-300 transition"
                        >
                          {copiedId === `hash_${evt.id}` ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                        </button>
                      )}
                    </div>
                  </td>

                  {/* Inspect Proof Button */}
                  <td className="py-3 px-4 text-right">
                    <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-400 hover:text-emerald-300">
                      <span>Proof</span>
                      <ExternalLink className="w-3 h-3" />
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination Footer */}
      <div className="p-3 border-t border-slate-800 bg-slate-950/60 flex items-center justify-between text-xs text-slate-400">
        <div>
          Showing {events.length > 0 ? offset + 1 : 0} to {Math.min(offset + limit, total)} of {total} records
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => onPageChange(Math.max(0, offset - limit))}
            disabled={offset === 0}
            className="p-1 rounded bg-slate-800 hover:bg-slate-700 disabled:opacity-40 disabled:pointer-events-none text-slate-300"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>
          <span>Page {currentPage} of {totalPages}</span>
          <button
            onClick={() => onPageChange(offset + limit)}
            disabled={offset + limit >= total}
            className="p-1 rounded bg-slate-800 hover:bg-slate-700 disabled:opacity-40 disabled:pointer-events-none text-slate-300"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
};
