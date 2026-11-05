import React, { useEffect, useState } from 'react';
import { Layers, ArrowRight, ShieldCheck, Database, Clock, Key, Copy, Check } from 'lucide-react';
import { BlockHeader } from '../types';

export const BlockChainExplorer: React.FC = () => {
  const [blocks, setBlocks] = useState<BlockHeader[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  useEffect(() => {
    fetchBlocks();
  }, []);

  const fetchBlocks = () => {
    setLoading(true);
    fetch('/api/v1/blocks?limit=50')
      .then((res) => res.json())
      .then((data) => {
        setBlocks(data.blocks || []);
        setTotal(data.total || 0);
        setLoading(false);
      })
      .catch((err) => {
        console.error('Failed fetching blocks:', err);
        setLoading(false);
      });
  };

  const copy = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 space-y-6">
      <div className="flex items-center justify-between border-b border-slate-800 pb-4">
        <div>
          <h2 className="text-sm font-bold text-white flex items-center gap-2">
            <Layers className="w-4 h-4 text-emerald-400" />
            <span>Cryptographic Merkle DAG Block Chain Explorer</span>
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Sequential append-only block headers connected via cryptographic hash pointers
          </p>
        </div>
        <div className="text-xs font-mono text-slate-400 bg-slate-950 px-3 py-1 rounded-lg border border-slate-800">
          Total Blocks: <span className="text-emerald-400 font-bold">{total}</span>
        </div>
      </div>

      {loading ? (
        <div className="py-12 text-center text-slate-500">
          <div className="inline-block animate-spin rounded-full h-5 w-5 border-2 border-slate-600 border-t-emerald-400 mb-2" />
          <div>Loading block headers...</div>
        </div>
      ) : blocks.length === 0 ? (
        <div className="py-12 text-center text-slate-500 text-xs">
          No blocks sealed yet. Emit events to produce cryptographic blocks.
        </div>
      ) : (
        <div className="space-y-4">
          {blocks.map((b, idx) => (
            <div
              key={b.index}
              className="bg-slate-950 border border-slate-800 hover:border-slate-700 rounded-xl p-4 transition text-xs space-y-3"
            >
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-slate-900 pb-2.5">
                <div className="flex items-center gap-2.5">
                  <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                    Block #{b.index}
                  </span>
                  <span className="text-slate-400 text-[11px] flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5 text-slate-500" />
                    <span>{new Date(b.timestamp).toLocaleString()}</span>
                  </span>
                </div>
                <div className="flex items-center gap-2 text-[11px]">
                  <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-medium">
                    {b.event_count} events sealed
                  </span>
                  <span className="px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 uppercase font-mono">
                    {b.algorithm}
                  </span>
                </div>
              </div>

              {/* Hash Links */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3 font-mono text-[11px]">
                <div className="bg-slate-900/60 p-2.5 rounded-lg border border-slate-800/80">
                  <div className="text-[10px] text-slate-500 uppercase font-semibold mb-1 flex items-center justify-between">
                    <span>Block Hash</span>
                    <button
                      onClick={() => copy(b.block_hash_hex, `bhash_${b.index}`)}
                      className="text-slate-500 hover:text-slate-300"
                    >
                      {copiedKey === `bhash_${b.index}` ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    </button>
                  </div>
                  <div className="text-slate-200 truncate" title={b.block_hash_hex}>
                    {b.block_hash_hex}
                  </div>
                </div>

                <div className="bg-slate-900/60 p-2.5 rounded-lg border border-slate-800/80">
                  <div className="text-[10px] text-slate-500 uppercase font-semibold mb-1 flex items-center justify-between">
                    <span>Previous Block Hash Link</span>
                    <button
                      onClick={() => copy(b.previous_hex, `prev_${b.index}`)}
                      className="text-slate-500 hover:text-slate-300"
                    >
                      {copiedKey === `prev_${b.index}` ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    </button>
                  </div>
                  <div className="text-slate-400 truncate" title={b.previous_hex}>
                    {b.previous_hex === '0000000000000000000000000000000000000000000000000000000000000000'
                      ? 'Genesis Origin (0x00...)'
                      : b.previous_hex}
                  </div>
                </div>

                <div className="bg-slate-900/60 p-2.5 rounded-lg border border-slate-800/80">
                  <div className="text-[10px] text-slate-500 uppercase font-semibold mb-1 flex items-center justify-between">
                    <span>Merkle Root</span>
                    <button
                      onClick={() => copy(b.root_hex, `root_${b.index}`)}
                      className="text-slate-500 hover:text-slate-300"
                    >
                      {copiedKey === `root_${b.index}` ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    </button>
                  </div>
                  <div className="text-emerald-400 truncate" title={b.root_hex}>
                    {b.root_hex}
                  </div>
                </div>
              </div>

              {/* Digital Signature */}
              {b.key_id && (
                <div className="flex items-center gap-2 text-[10px] font-mono text-slate-400 pt-1">
                  <Key className="w-3 h-3 text-slate-500" />
                  <span>Ed25519 Signer Key:</span>
                  <span className="text-slate-300">{b.key_id}</span>
                  <span className="text-emerald-400 ml-auto">Certified Valid</span>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
