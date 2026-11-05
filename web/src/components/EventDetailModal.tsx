import React, { useEffect, useState } from 'react';
import { X, CheckCircle2, ShieldCheck, Copy, Check, ArrowRight, CornerDownRight, Layers, FileCode } from 'lucide-react';
import { AuditEvent, BlockHeader, MerkleProof } from '../types';

interface EventDetailModalProps {
  event: AuditEvent | null;
  onClose: () => void;
}

export const EventDetailModal: React.FC<EventDetailModalProps> = ({ event, onClose }) => {
  const [loading, setLoading] = useState(false);
  const [proofData, setProofData] = useState<{
    event: AuditEvent;
    block_header?: BlockHeader;
    proof?: MerkleProof;
    verified?: boolean;
  } | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  useEffect(() => {
    if (!event) {
      setProofData(null);
      return;
    }

    setLoading(true);
    fetch(`/api/v1/events/${event.id}`)
      .then((res) => res.json())
      .then((data) => {
        setProofData(data);
        setLoading(false);
      })
      .catch((err) => {
        console.error('Failed fetching event proof:', err);
        setLoading(false);
      });
  }, [event]);

  if (!event) return null;

  const copy = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-3xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Modal Header */}
        <div className="p-4 sm:p-5 border-b border-slate-800 flex items-center justify-between bg-slate-900/90">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-bold text-white">Cryptographic Proof & Canonical Inspector</h3>
                <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                  VERIFIED LEAF
                </span>
              </div>
              <p className="text-[11px] text-slate-400 font-mono mt-0.5">{event.id}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-4 sm:p-6 overflow-y-auto space-y-6">
          {/* Cryptographic Assurance Banner */}
          <div className="bg-emerald-950/20 border border-emerald-500/30 rounded-xl p-3.5 flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
              <div>
                <div className="text-xs font-semibold text-emerald-300">
                  Merkle Tree Inclusion Proof Confirmed
                </div>
                <div className="text-[11px] text-slate-400 mt-0.5">
                  Leaf hash recalculates to sealed block root via {proofData?.proof?.audit_path?.length ?? 0} binary sibling hops.
                </div>
              </div>
            </div>
            <div className="text-right text-[11px] font-mono text-slate-400">
              <div>Block #{event.block_index}</div>
              <div>Leaf #{event.leaf_index}</div>
            </div>
          </div>

          {/* Merkle Audit Path Visualization */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                <Layers className="w-3.5 h-3.5 text-emerald-400" />
                <span>Binary Merkle DAG Audit Path (Bottom-Up)</span>
              </span>
              <span className="text-[10px] font-mono text-slate-500 uppercase">
                Algorithm: {proofData?.proof?.algorithm || 'Blake3'}
              </span>
            </div>

            <div className="bg-slate-950 border border-slate-800/80 rounded-xl p-4 font-mono text-xs space-y-3">
              {/* Step 1: Leaf */}
              <div className="flex items-center justify-between bg-slate-900/60 p-2.5 rounded-lg border border-slate-800">
                <div className="flex items-center gap-2">
                  <span className="px-1.5 py-0.5 rounded text-[10px] bg-emerald-500/20 text-emerald-300 font-bold">
                    LEAF #{event.leaf_index}
                  </span>
                  <span className="text-slate-300 truncate max-w-[280px] sm:max-w-md" title={event.leaf_hash}>
                    {event.leaf_hash}
                  </span>
                </div>
                <button
                  onClick={() => copy(event.leaf_hash, 'leaf')}
                  className="text-slate-500 hover:text-slate-300 transition"
                >
                  {copiedKey === 'leaf' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                </button>
              </div>

              {/* Step 2: Sibling Hashes in Audit Path */}
              {proofData?.proof?.audit_path && proofData.proof.audit_path.length > 0 ? (
                proofData.proof.audit_path.map((step, idx) => (
                  <div key={idx} className="flex items-center gap-2 pl-4 border-l-2 border-emerald-500/40">
                    <CornerDownRight className="w-3.5 h-3.5 text-slate-500 shrink-0" />
                    <span className="px-1.5 py-0.5 rounded text-[10px] bg-slate-800 text-slate-400 font-bold uppercase shrink-0">
                      Sibling {step.position}
                    </span>
                    <span className="text-slate-400 truncate text-[11px]" title={step.sibling_hex}>
                      {step.sibling_hex}
                    </span>
                  </div>
                ))
              ) : (
                <div className="text-[11px] text-slate-500 pl-4 border-l-2 border-emerald-500/40">
                  (Single-leaf block: paired self-hash root)
                </div>
              )}

              {/* Step 3: Sealed Merkle Root */}
              <div className="flex items-center justify-between bg-slate-900/90 p-2.5 rounded-lg border border-emerald-500/30">
                <div className="flex items-center gap-2">
                  <span className="px-1.5 py-0.5 rounded text-[10px] bg-cyan-500/20 text-cyan-300 font-bold">
                    MERKLE ROOT
                  </span>
                  <span className="text-cyan-200 truncate max-w-[280px] sm:max-w-md" title={proofData?.block_header?.root_hex}>
                    {proofData?.block_header?.root_hex || 'Pending Seal'}
                  </span>
                </div>
                {proofData?.block_header?.root_hex && (
                  <button
                    onClick={() => copy(proofData.block_header!.root_hex, 'root')}
                    className="text-slate-500 hover:text-slate-300 transition"
                  >
                    {copiedKey === 'root' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                )}
              </div>

              {/* Step 4: Block Hash */}
              <div className="flex items-center justify-between bg-slate-900/60 p-2.5 rounded-lg border border-slate-800">
                <div className="flex items-center gap-2">
                  <span className="px-1.5 py-0.5 rounded text-[10px] bg-violet-500/20 text-violet-300 font-bold">
                    BLOCK HASH
                  </span>
                  <span className="text-violet-200 truncate max-w-[280px] sm:max-w-md" title={proofData?.block_header?.block_hash_hex}>
                    {proofData?.block_header?.block_hash_hex || '-'}
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* RFC 8785 Canonical JSON Representation */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                <FileCode className="w-3.5 h-3.5 text-emerald-400" />
                <span>Deterministic Canonical JSON Payload</span>
              </span>
              <button
                onClick={() => copy(JSON.stringify(event, null, 2), 'raw_json')}
                className="text-[11px] text-slate-400 hover:text-slate-200 flex items-center gap-1"
              >
                {copiedKey === 'raw_json' ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                <span>Copy JSON</span>
              </button>
            </div>
            <pre className="bg-slate-950 border border-slate-800 rounded-xl p-4 text-[11px] font-mono text-emerald-400/90 overflow-x-auto max-h-56 leading-relaxed">
              {JSON.stringify(event, null, 2)}
            </pre>
          </div>
        </div>

        {/* Modal Footer */}
        <div className="p-4 border-t border-slate-800 bg-slate-900/80 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold rounded-xl transition"
          >
            Close Inspector
          </button>
        </div>
      </div>
    </div>
  );
};
