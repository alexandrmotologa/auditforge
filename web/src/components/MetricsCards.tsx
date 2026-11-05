import React from 'react';
import { FileText, Database, Building2, Cpu } from 'lucide-react';
import { LedgerStats } from '../types';

interface MetricsCardsProps {
  stats: LedgerStats | null;
}

export const MetricsCards: React.FC<MetricsCardsProps> = ({ stats }) => {
  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-slate-400">Total Certified Events</span>
          <FileText className="w-4 h-4 text-emerald-400" />
        </div>
        <div className="mt-2 text-2xl font-bold text-white tracking-tight">
          {stats ? stats.total_events.toLocaleString() : '0'}
        </div>
        <div className="mt-1 text-[11px] text-slate-400">
          Strict canonical serialization
        </div>
      </div>

      <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-slate-400">Sealed Merkle Blocks</span>
          <Database className="w-4 h-4 text-cyan-400" />
        </div>
        <div className="mt-2 text-2xl font-bold text-white tracking-tight">
          {stats ? stats.total_blocks.toLocaleString() : '0'}
        </div>
        <div className="mt-1 text-[11px] text-slate-400">
          Append-only WAL durability
        </div>
      </div>

      <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-slate-400">Active Tenant Orgs</span>
          <Building2 className="w-4 h-4 text-violet-400" />
        </div>
        <div className="mt-2 text-2xl font-bold text-white tracking-tight">
          {stats ? stats.tenant_count.toLocaleString() : '0'}
        </div>
        <div className="mt-1 text-[11px] text-slate-400">
          Multi-tenant partitioned index
        </div>
      </div>

      <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-slate-400">Cryptographic Engine</span>
          <Cpu className="w-4 h-4 text-amber-400" />
        </div>
        <div className="mt-2 text-lg font-bold text-white tracking-tight truncate uppercase">
          {stats?.algorithm || 'Blake3'} + SHA-256
        </div>
        <div className="mt-1 text-[11px] text-slate-400">
          Ed25519 Block Signing
        </div>
      </div>
    </div>
  );
};
