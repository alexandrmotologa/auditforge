export interface Actor {
  id: string;
  name?: string;
  email?: string;
  role?: string;
  metadata?: Record<string, unknown>;
}

export interface Target {
  id: string;
  type: string;
  name?: string;
  metadata?: Record<string, unknown>;
}

export interface AuditEventInput {
  id?: string;
  timestamp?: string;
  tenant_id?: string;
  actor: Actor;
  action: string;
  target: Target;
  metadata?: Record<string, unknown>;
}

export interface ClientConfig {
  endpoint?: string;
  apiKey?: string;
  tenantId?: string;
  batchSize?: number;
  flushIntervalMs?: number;
}

export class AuditForgeClient {
  private endpoint: string;
  private apiKey?: string;
  private tenantId?: string;
  private batchSize: number;
  private flushIntervalMs: number;
  private buffer: AuditEventInput[] = [];
  private timer: ReturnType<typeof setInterval> | null = null;

  constructor(config: ClientConfig = {}) {
    this.endpoint = (config.endpoint || 'http://localhost:8080').replace(/\/$/, '');
    this.apiKey = config.apiKey;
    this.tenantId = config.tenantId;
    this.batchSize = config.batchSize || 100;
    this.flushIntervalMs = config.flushIntervalMs || 250;

    this.timer = setInterval(() => {
      this.flush().catch((err) => console.error('[AuditForge SDK] Flush error:', err));
    }, this.flushIntervalMs);
  }

  public emit(event: AuditEventInput): void {
    if (!event.tenant_id && this.tenantId) {
      event.tenant_id = this.tenantId;
    }
    if (!event.timestamp) {
      event.timestamp = new Date().toISOString();
    }

    this.buffer.push(event);
    if (this.buffer.length >= this.batchSize) {
      this.flush().catch((err) => console.error('[AuditForge SDK] Flush error:', err));
    }
  }

  public async emitSync(event: AuditEventInput): Promise<{ status: string; event_id: string; leaf_hash: string }> {
    if (!event.tenant_id && this.tenantId) {
      event.tenant_id = this.tenantId;
    }
    if (!event.timestamp) {
      event.timestamp = new Date().toISOString();
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    const res = await fetch(`${this.endpoint}/api/v1/events?sync=true`, {
      method: 'POST',
      headers,
      body: JSON.stringify(event),
    });

    if (!res.ok) {
      throw new Error(`AuditForge sync ingestion failed with status ${res.status}`);
    }

    return (await res.json()) as { status: string; event_id: string; leaf_hash: string };
  }

  public async flush(): Promise<void> {
    if (this.buffer.length === 0) return;

    const batch = [...this.buffer];
    this.buffer = [];

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    const payload = {
      tenant_id: this.tenantId,
      events: batch,
    };

    const res = await fetch(`${this.endpoint}/api/v1/events/batch`, {
      method: 'POST',
      headers,
      body: JSON.stringify(payload),
    });

    if (!res.ok) {
      // Put failed events back in buffer
      this.buffer.unshift(...batch);
      throw new Error(`AuditForge batch flush failed with status ${res.status}`);
    }
  }

  public async close(): Promise<void> {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
    await this.flush();
  }
}
