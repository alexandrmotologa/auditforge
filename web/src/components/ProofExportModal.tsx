import React, { useState } from 'react';
import { X, Download, ShieldCheck, FileCheck2, Terminal } from 'lucide-react';
import { LedgerStats } from '../types';

interface ProofExportModalProps {
  isOpen: boolean;
  onClose: () => void;
  stats: LedgerStats | null;
}

export const ProofExportModal: React.FC<ProofExportModalProps> = ({ isOpen, onClose, stats }) => {
  const [startBlock, setStartBlock] = useState(0);
  const [endBlock, setEndBlock] = useState(stats?.total_blocks ? Math.max(0, stats.total_blocks - 1) : 0);
  const [isExporting, setIsExporting] = useState(false);

  if (!isOpen) return null;

  const handleDownloadProof = () => {
    setIsExporting(true);
    const link = document.createElement('a');
    link.href = `/api/v1/export?from=${startBlock}&to=${endBlock}`;
    link.download = `auditforge_compliance_export_${Date.now()}.proof`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    setTimeout(() => setIsExporting(false), 1000);
  };

  const handleDownloadReport = () => {
    const link = document.createElement('a');
    link.href = `/api/v1/report?from=${startBlock}&to=${endBlock}`;
    link.download = `auditforge_compliance_certificate_${Date.now()}.html`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden">
        <div className="p-5 border-b border-slate-800 flex items-center justify-between bg-slate-900/90">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <Download className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-white">Export Cryptographic Proof Package</h3>
              <p className="text-xs text-slate-400">Self-contained .proof bundle for compliance auditors</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-5">
          <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 text-xs text-slate-300 space-y-2">
            <div className="font-semibold text-slate-200 flex items-center gap-2">
              <ShieldCheck className="w-4 h-4 text-emerald-400" />
              <span>Offline Verification Guarantees</span>
            </div>
            <p className="text-slate-400 leading-relaxed">
              The exported package contains canonical JSON events, Merkle audit trees, and digital signatures.
              Auditors can run verification completely offline without database credentials or network access:
            </p>
            <div className="bg-slate-900 border border-slate-800 rounded-lg p-2.5 font-mono text-[11px] text-emerald-400 flex items-center gap-2">
              <Terminal className="w-3.5 h-3.5 text-slate-500" />
              <span>auditverify verify export.proof</span>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">From Block</label>
              <input
                type="number"
                min="0"
                value={startBlock}
                onChange={(e) => setStartBlock(parseInt(e.target.value) || 0)}
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-200 focus:outline-none focus:border-emerald-500/50"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">To Block</label>
              <input
                type="number"
                min="0"
                value={endBlock}
                onChange={(e) => setEndBlock(parseInt(e.target.value) || 0)}
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-200 focus:outline-none focus:border-emerald-500/50"
              />
            </div>
          </div>
        </div>

        <div className="p-4 border-t border-slate-800 bg-slate-900/80 flex items-center justify-between">
          <button
            onClick={handleDownloadReport}
            className="flex items-center gap-1.5 px-3 py-2 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition"
          >
            <FileCheck2 className="w-3.5 h-3.5 text-cyan-400" />
            <span>HTML Certificate</span>
          </button>

          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white transition"
            >
              Cancel
            </button>
            <button
              onClick={handleDownloadProof}
              disabled={isExporting}
              className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold bg-emerald-600 hover:bg-emerald-500 text-white transition shadow-sm shadow-emerald-950"
            >
              <Download className="w-3.5 h-3.5" />
              <span>{isExporting ? 'Generating...' : 'Download .proof'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
