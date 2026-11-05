import React from 'react';
import { ShieldCheck, ShieldAlert, Download, RefreshCw, Radio, Plus, FlaskConical } from 'lucide-react';
import { LedgerStats } from '../types';

interface HeaderProps {
  stats: LedgerStats | null;
  isStreaming: boolean;
  onToggleStream: () => void;
  onVerify: () => void;
  onOpenExport: () => void;
  onOpenTamper: () => void;
  onInjectTestEvent: () => void;
  isVerifying: boolean;
  isInjecting: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  stats,
  isStreaming,
  onToggleStream,
  onVerify,
  onOpenExport,
  onOpenTamper,
  onInjectTestEvent,
  isVerifying,
  isInjecting,
}) => {
  return (
    <header className="border-b border-slate-800 bg-slate-900/60 backdrop-blur-md sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 shadow-inner">
            <ShieldCheck className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-lg tracking-tight text-white">AuditForge</span>
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                v1.0 • SOC 2 & HIPAA
              </span>
            </div>
            <p className="text-xs text-slate-400 hidden sm:block">
              Tamper-Evident Immutable Audit Log & Merkle DAG Ledger
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2 sm:gap-3">
          {/* Real-time SSE indicator */}
          <button
            onClick={onToggleStream}
            className={`flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-xs font-medium border transition-colors ${
              isStreaming
                ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
                : 'bg-slate-800/80 border-slate-700 text-slate-400'
            }`}
            title={isStreaming ? "Live SSE stream active (Click to pause)" : "Live stream paused (Click to resume)"}
          >
            <span className="relative flex h-2 w-2">
              {isStreaming && (
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
              )}
              <span className={`relative inline-flex rounded-full h-2 w-2 ${isStreaming ? 'bg-emerald-500' : 'bg-slate-500'}`} />
            </span>
            <span className="hidden md:inline">{isStreaming ? "Live Stream" : "Stream Paused"}</span>
          </button>

          {/* Quick test event injector */}
          <button
            onClick={onInjectTestEvent}
            disabled={isInjecting}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition"
            title="Send sample compliance event via REST"
          >
            <Plus className="w-3.5 h-3.5 text-emerald-400" />
            <span className="hidden md:inline">Emit Event</span>
          </button>

          {/* Tamper simulation tester */}
          <button
            onClick={onOpenTamper}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border transition ${
              stats?.tamper_simulation
                ? 'bg-rose-500/20 border-rose-500/40 text-rose-300 animate-pulse'
                : 'bg-slate-800/80 hover:bg-slate-700 text-slate-300 border-slate-700'
            }`}
            title="Demonstration: Inject malicious modification to test immediate tamper detection"
          >
            <FlaskConical className="w-3.5 h-3.5 text-amber-400" />
            <span className="hidden md:inline">Tamper Demo</span>
          </button>

          {/* On-demand Verify */}
          <button
            onClick={onVerify}
            disabled={isVerifying}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition"
            title="Recalculate complete Merkle DAG and block hash chains"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-emerald-400 ${isVerifying ? 'animate-spin' : ''}`} />
            <span className="hidden md:inline">Verify All</span>
          </button>

          {/* Export Proof Package */}
          <button
            onClick={onOpenExport}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-emerald-600 hover:bg-emerald-500 text-white shadow-sm shadow-emerald-950 transition"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export .proof</span>
          </button>
        </div>
      </div>
    </header>
  );
};
