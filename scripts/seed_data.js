const http = require('http');

const events = [
  {
    tenant_id: 'org_acme_corp',
    actor: {
      id: 'usr_alexander',
      name: 'Alexandr Motologa',
      email: 'alex@example.com',
      role: 'Principal Security Architect'
    },
    action: 'role.permission.grant',
    target: {
      id: 'role_soc2_auditor',
      type: 'iam_role',
      name: 'SOC 2 External Compliance Auditor'
    },
    metadata: {
      ip_address: '192.168.1.104',
      user_agent: 'AuditForge-SDK/1.0.0 (Go 1.27)',
      environment: 'production',
      framework: 'SOC 2 Type II - CC6.1 Logical Access Controls',
      previous_state: {
        role: 'soc2_auditor',
        mfa_required: true,
        session_duration_minutes: 60,
        permissions: ['read:audit_summary']
      },
      new_state: {
        role: 'soc2_auditor',
        mfa_required: true,
        session_duration_minutes: 120,
        permissions: ['read:audit_summary', 'read:merkle_ledger', 'export:proof_bundle', 'verify:offline_tree']
      }
    }
  },
  {
    tenant_id: 'org_acme_corp',
    actor: {
      id: 'usr_sarah',
      name: 'Sarah Chen',
      email: 'sarah.chen@acme.internal',
      role: 'Head of Infrastructure Security'
    },
    action: 'database.credential.rotate',
    target: {
      id: 'db_vault_prod_primary',
      type: 'vault_secret',
      name: 'PostgreSQL Primary Shard Credential'
    },
    metadata: {
      ip_address: '10.240.12.8',
      user_agent: 'HashiCorp-Vault-Sync/2.4',
      environment: 'production',
      framework: 'HIPAA Security Rule § 164.312(a)(2)(iv)',
      rotation_policy: '90_day_crypto_rotation',
      previous_state: {
        key_version: 14,
        cipher_suite: 'AES-256-GCM',
        active_connections: 420
      },
      new_state: {
        key_version: 15,
        cipher_suite: 'AES-256-GCM',
        active_connections: 420,
        zero_downtime_sync: true
      }
    }
  },
  {
    tenant_id: 'org_health_first',
    actor: {
      id: 'usr_elena',
      name: 'Dr. Elena Rostova',
      email: 'elena.rostova@healthfirst.med',
      role: 'Chief Medical Officer'
    },
    action: 'patient.record.access',
    target: {
      id: 'rec_pt_8839210',
      type: 'phi_record',
      name: 'HIPAA Protected Health Information Record #8839210'
    },
    metadata: {
      ip_address: '172.16.4.15',
      user_agent: 'Epic-EHR-Connector/4.12',
      environment: 'production',
      framework: 'HIPAA Privacy Rule § 164.502 Minimum Necessary Access',
      department: 'Oncology Research',
      justification: 'Approved IRB Clinical Trial Protocol #2026-B9'
    }
  },
  {
    tenant_id: 'org_acme_corp',
    actor: {
      id: 'usr_marcus',
      name: 'Marcus Vance',
      email: 'marcus.v@acme.internal',
      role: 'DevOps Lead'
    },
    action: 'api_key.create',
    target: {
      id: 'key_ingest_siem_prod',
      type: 'api_key',
      name: 'SIEM High-Throughput Ingestion Token'
    },
    metadata: {
      ip_address: '10.240.8.91',
      user_agent: 'Terraform/1.9.2 (+provider-auditforge/1.0)',
      environment: 'production',
      framework: 'SOC 2 CC6.8 Threat Detection & Logging',
      allowed_ips: ['10.240.0.0/16'],
      scopes: ['events:write', 'ledger:read']
    }
  },
  {
    tenant_id: 'org_fintech_vault',
    actor: {
      id: 'usr_julian',
      name: 'Julian Thorne',
      email: 'julian@fintechvault.io',
      role: 'Compliance Director'
    },
    action: 'compliance.report.export',
    target: {
      id: 'pkg_soc2_q3_certified',
      type: 'evidence_bundle',
      name: 'Q3 2026 Cryptographic Ledger Audit Archive'
    },
    metadata: {
      ip_address: '192.168.10.42',
      user_agent: 'AuditForge-Cockpit/1.0 (Web/React 19)',
      environment: 'production',
      framework: 'AICPA SOC 2 Trust Services Criteria',
      certificate_seal: 'Ed25519-Signed Blake3 Merkle Root'
    }
  },
  {
    tenant_id: 'org_health_first',
    actor: {
      id: 'usr_david',
      name: 'David Miller',
      email: 'david.m@healthfirst.med',
      role: 'Security Engineer'
    },
    action: 'mfa.factor.enroll',
    target: {
      id: 'usr_elena',
      type: 'user_identity',
      name: 'FIDO2 WebAuthn Hardware Security Key'
    },
    metadata: {
      ip_address: '172.16.4.15',
      user_agent: 'Chrome/133.0 (Windows 11)',
      environment: 'production',
      framework: 'HIPAA Security Rule § 164.312(d)',
      hardware_vendor: 'Yubico YubiKey 5C NFC'
    }
  },
  {
    tenant_id: 'org_acme_corp',
    actor: {
      id: 'usr_alexander',
      name: 'Alexandr Motologa',
      email: 'alex@example.com',
      role: 'Principal Security Architect'
    },
    action: 'policy.security.update',
    target: {
      id: 'pol_zero_trust_ingress',
      type: 'network_policy',
      name: 'Zero-Trust mTLS Boundary Ruleset'
    },
    metadata: {
      ip_address: '192.168.1.104',
      user_agent: 'AuditForge-CLI/1.0',
      environment: 'production',
      framework: 'SOC 2 CC6.6 Boundary Protection',
      previous_state: {
        mtls_mode: 'permissive',
        cipher_suites: ['TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256', 'TLS_AES_256_GCM_SHA384']
      },
      new_state: {
        mtls_mode: 'strict_enforced',
        cipher_suites: ['TLS_AES_256_GCM_SHA384', 'TLS_CHACHA20_POLY1305_SHA256']
      }
    }
  },
  {
    tenant_id: 'org_fintech_vault',
    actor: {
      id: 'usr_rachel',
      name: 'Rachel Adams',
      email: 'rachel@fintechvault.io',
      role: 'Chief Risk Officer'
    },
    action: 'settlement.rule.approve',
    target: {
      id: 'rule_instant_ach_limits',
      type: 'financial_policy',
      name: 'Instant ACH Clearing Threshold Matrix'
    },
    metadata: {
      ip_address: '192.168.10.15',
      user_agent: 'AuditForge-Cockpit/1.0',
      environment: 'production',
      framework: 'PCI-DSS v4.0 Requirement 10.2'
    }
  }
];

function sendEvent(evt) {
  return new Promise((resolve, reject) => {
    const data = JSON.stringify(evt);
    const req = http.request(
      {
        hostname: 'localhost',
        port: 8080,
        path: '/api/v1/events?sync=true',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(data),
          'X-API-Key': 'default-dev-key',
        },
      },
      (res) => {
        let body = '';
        res.on('data', (chunk) => (body += chunk));
        res.on('end', () => {
          if (res.statusCode >= 200 && res.statusCode < 300) {
            resolve(JSON.parse(body));
          } else {
            reject(new Error(`Status ${res.statusCode}: ${body}`));
          }
        });
      }
    );
    req.on('error', reject);
    req.write(data);
    req.end();
  });
}

async function run() {
  console.log(`Seeding ${events.length} enterprise audit events...`);
  for (let i = 0; i < events.length; i++) {
    const res = await sendEvent(events[i]);
    console.log(`[${i + 1}/${events.length}] Seeded: ${events[i].action} (ID: ${res.event_id}, Block: ${res.block_index})`);
    // brief delay to ensure distinct microsecond timestamps
    await new Promise((r) => setTimeout(r, 60));
  }
  console.log('Finished seeding real compliance events!');
}

run().catch(console.error);
