import React, { useEffect, useState, useCallback, useRef } from 'react';
import { Header } from './components/Header';
import { IntegrityShield } from './components/IntegrityShield';
import { MetricsCards } from './components/MetricsCards';
import { AuditFeed } from './components/AuditFeed';
import { EventDetailModal } from './components/EventDetailModal';
import { ProofExportModal } from './components/ProofExportModal';
import { TamperSimulatorModal } from './components/TamperSimulatorModal';
import { BlockChainExplorer } from './components/BlockChainExplorer';
import { AuditEvent, LedgerStats, VerificationResult } from './types';
import { ListFilter, Layers } from 'lucide-react';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'events' | 'blockchain'>('events');
  const [stats, setStats] = useState<LedgerStats | null>(null);
  const [verResult, setVerResult] = useState<VerificationResult | null>(null);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [limit] = useState(25);
  const [offset, setOffset] = useState(0);
  const [filters, setFilters] = useState<{ tenant_id?: string; action?: string; q?: string }>({});

  const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null);
  const [isExportOpen, setIsExportOpen] = useState(false);
  const [isTamperOpen, setIsTamperOpen] = useState(false);

  const [isVerifying, setIsVerifying] = useState(false);
  const [isInjecting, setIsInjecting] = useState(false);
  const [isStreaming, setIsStreaming] = useState(true);
  const [isLoadingEvents, setIsLoadingEvents] = useState(false);

  const sseRef = useRef<EventSource | null>(null);

  // Fetch High-Level Stats
  const fetchStats = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/stats');
      if (res.ok) {
        const data = await res.json();
        setStats(data);
      }
    } catch (err) {
      console.error('Error fetching stats:', err);
    }
  }, []);

  // Fetch Paginated Events
  const fetchEvents = useCallback(async (currentOffset: number, currentFilters: typeof filters) => {
    setIsLoadingEvents(true);
    try {
      const params = new URLSearchParams();
      params.set('limit', limit.toString());
      params.set('offset', currentOffset.toString());
      if (currentFilters.tenant_id) params.set('tenant_id', currentFilters.tenant_id);
      if (currentFilters.action) params.set('action', currentFilters.action);
      if (currentFilters.q) params.set('q', currentFilters.q);

      const res = await fetch(`/api/v1/events?${params.toString()}`);
      if (res.ok) {
        const data = await res.json();
        setEvents(data.events || []);
        setTotal(data.total || 0);
      }
    } catch (err) {
      console.error('Error fetching events:', err);
    } finally {
      setIsLoadingEvents(false);
    }
  }, [limit]);

  // Execute On-Demand Ledger Verification
  const runVerify = useCallback(async () => {
    setIsVerifying(true);
    try {
      const res = await fetch('/api/v1/verify');
      if (res.ok) {
        const data = await res.json();
        setVerResult(data);
      }
      await fetchStats();
    } catch (err) {
      console.error('Error verifying ledger:', err);
    } finally {
      setIsVerifying(false);
    }
  }, [fetchStats]);

  // Initial Load
  useEffect(() => {
    fetchStats();
    fetchEvents(offset, filters);
    runVerify();
  }, [fetchStats, fetchEvents, runVerify]);

  // SSE Real-Time Stream Setup
  useEffect(() => {
    if (!isStreaming) {
      if (sseRef.current) {
        sseRef.current.close();
        sseRef.current = null;
      }
      return;
    }

    const sse = new EventSource('/api/v1/stream');
    sseRef.current = sse;

    sse.addEventListener('block_sealed', () => {
      fetchStats();
      fetchEvents(offset, filters);
    });

    sse.addEventListener('event_received', () => {
      fetchEvents(offset, filters);
    });

    sse.onerror = () => {
      // Reconnect handled automatically by EventSource
    };

    return () => {
      sse.close();
    };
  }, [isStreaming, offset, filters, fetchStats, fetchEvents]);

  // Emit Realistic Enterprise Sample Event
  const handleInjectTestEvent = async () => {
    setIsInjecting(true);
    const sampleActions = [
      'user.permission.grant',
      'database.credential.rotate',
      'customer.data.export',
      'api_key.create',
      'mfa.factor.enroll',
      'role.permission.revoke',
    ];
    const sampleActors = [
      { id: 'usr_alexander', name: 'Alexandr Motologa', email: 'alex@example.com', role: 'Security Architect' },
      { id: 'usr_sarah', name: 'Sarah Chen', email: 'sarah@example.com', role: 'Compliance Lead' },
      { id: 'usr_david', name: 'David Miller', email: 'david@ops.internal', role: 'Site Reliability Engineer' },
    ];
    const sampleTargets = [
      { id: 'cred_prod_readwrite', type: 'vault_secret', name: 'Production Database Credential' },
      { id: 'export_soc2_q3', type: 'compliance_report', name: 'Q3 Financial Audit Package' },
      { id: 'api_ingest_token', type: 'api_token', name: 'Enterprise Ingestion Token' },
      { id: 'auth_okta_gateway', type: 'identity_provider', name: 'Corporate Okta Federation' },
    ];

    const randomAction = sampleActions[Math.floor(Math.random() * sampleActions.length)];
    const randomActor = sampleActors[Math.floor(Math.random() * sampleActors.length)];
    const randomTarget = sampleTargets[Math.floor(Math.random() * sampleTargets.length)];

    const payload = {
      tenant_id: 'org_acme_corp',
      actor: randomActor,
      action: randomAction,
      target: randomTarget,
      metadata: {
        ip_address: `192.168.1.${Math.floor(Math.random() * 250) + 1}`,
        user_agent: 'AuditForge-SDK/1.0',
        environment: 'production',
        previous_state: {
          role: 'viewer',
          mfa_enforced: false,
          permissions: ['read:public'],
        },
        new_state: {
          role: 'security_auditor',
          mfa_enforced: true,
          permissions: ['read:public', 'read:audit_ledger', 'export:proof_bundle'],
        },
      },
    };

    try {
      await fetch('/api/v1/events?sync=true', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': 'default-dev-key',
        },
        body: JSON.stringify(payload),
      });

      await fetchStats();
      await fetchEvents(offset, filters);
    } catch (err) {
      console.error('Failed injecting test event:', err);
    } finally {
      setIsInjecting(false);
    }
  };

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
    fetchEvents(newOffset, filters);
  };

  const handleFilterChange = (newFilters: typeof filters) => {
    setFilters(newFilters);
    setOffset(0);
    fetchEvents(0, newFilters);
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col">
      {/* Top Navigation */}
      <Header
        stats={stats}
        isStreaming={isStreaming}
        onToggleStream={() => setIsStreaming(!isStreaming)}
        onVerify={runVerify}
        onOpenExport={() => setIsExportOpen(true)}
        onOpenTamper={() => setIsTamperOpen(true)}
        onInjectTestEvent={handleInjectTestEvent}
        isVerifying={isVerifying}
        isInjecting={isInjecting}
      />

      {/* Main Body */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
        {/* Dynamic Cryptographic Shield Banner */}
        <IntegrityShield
          verResult={verResult}
          stats={stats}
          isVerifying={isVerifying}
          onVerify={runVerify}
        />

        {/* Operational Metrics Cards */}
        <MetricsCards stats={stats} />

        {/* View Switcher Tabs */}
        <div className="flex items-center justify-between border-b border-slate-800 pb-2">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setActiveTab('events')}
              className={`flex items-center gap-2 px-3.5 py-1.5 rounded-xl text-xs font-semibold transition ${
                activeTab === 'events'
                  ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                  : 'bg-slate-900/60 text-slate-400 hover:text-slate-200 border border-slate-800'
              }`}
            >
              <ListFilter className="w-3.5 h-3.5" />
              <span>Real-Time Audit Event Stream</span>
            </button>

            <button
              onClick={() => setActiveTab('blockchain')}
              className={`flex items-center gap-2 px-3.5 py-1.5 rounded-xl text-xs font-semibold transition ${
                activeTab === 'blockchain'
                  ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                  : 'bg-slate-900/60 text-slate-400 hover:text-slate-200 border border-slate-800'
              }`}
            >
              <Layers className="w-3.5 h-3.5" />
              <span>Block Chain & Merkle DAG Explorer</span>
            </button>
          </div>
        </div>

        {/* Tab Content */}
        {activeTab === 'events' ? (
          <AuditFeed
            events={events}
            total={total}
            limit={limit}
            offset={offset}
            onPageChange={handlePageChange}
            onFilterChange={handleFilterChange}
            onSelectEvent={(evt) => setSelectedEvent(evt)}
            isLoading={isLoadingEvents}
          />
        ) : (
          <BlockChainExplorer />
        )}
      </main>

      {/* Modals */}
      <EventDetailModal
        event={selectedEvent}
        onClose={() => setSelectedEvent(null)}
      />

      <ProofExportModal
        isOpen={isExportOpen}
        onClose={() => setIsExportOpen(false)}
        stats={stats}
      />

      <TamperSimulatorModal
        isOpen={isTamperOpen}
        onClose={() => setIsTamperOpen(false)}
        stats={stats}
        onRefresh={() => {
          runVerify();
          fetchStats();
        }}
      />

      {/* Footer */}
      <footer className="border-t border-slate-900 py-6 text-center text-xs text-slate-400">
        <p>
          AuditForge • Built with Go 1.27 + React 19 • Blake3 Merkle DAG & RFC 8785 Canonical Ledger • SOC 2 & HIPAA Ready
        </p>
      </footer>
    </div>
  );
};
