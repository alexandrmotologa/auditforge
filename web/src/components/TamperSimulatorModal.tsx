import React, { useState } from 'react';
import { X, FlaskConical, AlertTriangle, ShieldCheck, ShieldAlert, CheckCircle2, RotateCcw } from 'lucide-react';
import { LedgerStats } from '../types';

interface TamperSimulatorModalProps {
  isOpen: boolean;
  onClose: () => void;
  stats: LedgerStats | null;
  onRefresh: () => void;
}

export const TamperSimulatorModal: React.FC<TamperSimulatorModalProps> = ({
  isOpen,
  onClose,
  stats,
  onRefresh,
}) => {
  const [loading, setLoading] = useState(false);
  const isSimulated = stats?.tamper_simulation ?? false;

  if (!isOpen) return null;

  const handleToggleTamper = async (enable: boolean) => {
    setLoading(true);
    try {
      await fetch('/api/v1/tamper-simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: enable ? 'enable' : 'disable',
          block_index: 0,
        }),
      });
      onRefresh();
    } catch (err) {
      console.error('Tamper simulation error:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden">
        <div className="p-5 border-b border-slate-800 flex items-center justify-between bg-slate-900/90">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400">
              <FlaskConical className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-white">Auditor Demonstration Lab: Tamper Detection</h3>
              <p className="text-xs text-slate-400">Simulate malicious database modification</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 text-xs text-slate-300 leading-relaxed space-y-2">
            <div className="font-semibold text-white flex items-center gap-1.5">
              <AlertTriangle className="w-4 h-4 text-amber-400" />
              <span>Why Mutable SQL Tables Fail Audits</span>
            </div>
            <p className="text-slate-400">
              In standard PostgreSQL or MySQL setups, a privileged DBA or compromised server credential can silently execute:
            </p>
            <div className="bg-slate-900 border border-slate-800 rounded p-2 font-mono text-[11px] text-rose-300">
              UPDATE audit_logs SET action = 'harmless.read' WHERE id = 'evt_breach';
            </div>
            <p className="text-slate-400">
              In AuditForge, altering even 1 bit in a past event invalidates its Merkle leaf, breaks the block's Merkle root, and severs every subsequent block's cryptographic hash link.
            </p>
          </div>

          {/* Current Simulation Status */}
          <div
            className={`p-4 rounded-xl border flex items-center justify-between transition ${
              isSimulated
                ? 'bg-rose-950/30 border-rose-500/50 text-rose-300'
                : 'bg-emerald-950/20 border-emerald-500/30 text-emerald-300'
            }`}
          >
            <div className="flex items-center gap-3">
              {isSimulated ? (
                <ShieldAlert className="w-6 h-6 text-rose-400 shrink-0 animate-pulse" />
              ) : (
                <ShieldCheck className="w-6 h-6 text-emerald-400 shrink-0" />
              )}
              <div>
                <div className="text-xs font-bold uppercase tracking-wider">
                  {isSimulated ? 'Active Tamper Injected' : 'Ledger In Normal Immutable Mode'}
                </div>
                <div className="text-[11px] text-slate-400 mt-0.5">
                  {isSimulated
                    ? 'Block #0 event modified. Verification will immediately fail.'
                    : 'All blocks sealed and verified against continuous Merkle hash chain.'}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="p-4 border-t border-slate-800 bg-slate-900/80 flex items-center justify-between">
          <button
            onClick={onClose}
            className="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white transition"
          >
            Close Lab
          </button>

          {isSimulated ? (
            <button
              onClick={() => handleToggleTamper(false)}
              disabled={loading}
              className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold bg-emerald-600 hover:bg-emerald-500 text-white transition shadow-sm shadow-emerald-950"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              <span>Restore Ledger Integrity</span>
            </button>
          ) : (
            <button
              onClick={() => handleToggleTamper(true)}
              disabled={loading}
              className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold bg-rose-600 hover:bg-rose-500 text-white transition shadow-sm shadow-rose-950"
            >
              <FlaskConical className="w-3.5 h-3.5" />
              <span>Inject Simulated Tamper</span>
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
