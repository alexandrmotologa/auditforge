package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

// GenerateComplianceHTMLReport creates an auditor-ready HTML compliance assurance certificate
func GenerateComplianceHTMLReport(manifest *ProofManifest, chain []*ledger.BlockHeader, verResult *ledger.VerificationResult) string {
	statusClass := "status-valid"
	statusText := "VERIFIED IMMUTABLE & TAMPER-FREE"
	if !verResult.Valid {
		statusClass = "status-invalid"
		statusText = fmt.Sprintf("INTEGRITY VIOLATION DETECTED (Block #%v)", verResult.TamperedBlockIndex)
	}

	var rows strings.Builder
	for _, b := range chain {
		rows.WriteString(fmt.Sprintf(`
		<tr>
			<td>#%d</td>
			<td class="hash">%s...</td>
			<td class="hash">%s...</td>
			<td class="hash">%s...</td>
			<td>%d</td>
			<td>%s</td>
		</tr>`,
			b.Index,
			truncate(b.BlockHashHex, 16),
			truncate(b.PreviousHex, 16),
			truncate(b.RootHex, 16),
			b.EventCount,
			b.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC"),
		))
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>AuditForge Cryptographic Compliance Assurance Certificate</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 40px; color: #1e293b; background: #f8fafc; }
  .cert-container { max-width: 900px; margin: 0 auto; background: #ffffff; border: 1px solid #e2e8f0; border-radius: 12px; padding: 40px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1); }
  .header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #e2e8f0; padding-bottom: 20px; }
  .title { font-size: 24px; font-weight: 700; color: #0f172a; margin: 0; }
  .subtitle { font-size: 14px; color: #64748b; margin-top: 4px; }
  .badge { padding: 8px 16px; border-radius: 9999px; font-weight: 700; font-size: 13px; text-transform: uppercase; letter-spacing: 0.05em; }
  .status-valid { background: #dcfce7; color: #166534; border: 1px solid #86efac; }
  .status-invalid { background: #fee2e2; color: #991b1b; border: 1px solid #fca5a5; }
  .grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 20px; margin: 30px 0; }
  .card { background: #f1f5f9; padding: 16px; border-radius: 8px; }
  .card-label { font-size: 11px; text-transform: uppercase; color: #64748b; font-weight: 600; }
  .card-value { font-size: 16px; font-weight: 600; color: #0f172a; margin-top: 4px; word-break: break-all; }
  .hash { font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace; font-size: 12px; }
  table { width: 100%%; border-collapse: collapse; margin-top: 20px; font-size: 13px; }
  th { text-align: left; padding: 10px; background: #f8fafc; border-bottom: 2px solid #cbd5e1; color: #475569; }
  td { padding: 10px; border-bottom: 1px solid #e2e8f0; }
  .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #e2e8f0; font-size: 12px; color: #94a3b8; display: flex; justify-content: space-between; }
</style>
</head>
<body>
<div class="cert-container">
  <div class="header">
    <div>
      <h1 class="title">Cryptographic Compliance Ledger Certificate</h1>
      <div class="subtitle">SOC 2 Type II, ISO 27001 & HIPAA Append-Only Proof Verification</div>
    </div>
    <div class="badge %s">%s</div>
  </div>

  <div class="grid">
    <div class="card">
      <div class="card-label">Verification Timestamp</div>
      <div class="card-value">%s</div>
    </div>
    <div class="card">
      <div class="card-label">Hashing Algorithm & Specification</div>
      <div class="card-value">%s (RFC 8785 Canonical JSON)</div>
    </div>
    <div class="card">
      <div class="card-label">Total Validated Blocks</div>
      <div class="card-value">%d blocks (Range: #%d to #%d)</div>
    </div>
    <div class="card">
      <div class="card-label">Total Verified Events</div>
      <div class="card-value">%d events</div>
    </div>
    <div class="card">
      <div class="card-label">Ledger Head Block Hash</div>
      <div class="card-value hash">%s</div>
    </div>
    <div class="card">
      <div class="card-label">Compliance Signer Key ID</div>
      <div class="card-value hash">%s</div>
    </div>
  </div>

  <h3>Cryptographic Block Chain Integrity Table</h3>
  <table>
    <thead>
      <tr>
        <th>Block</th>
        <th>Block Hash</th>
        <th>Previous Hash</th>
        <th>Merkle Root</th>
        <th>Events</th>
        <th>Timestamp</th>
      </tr>
    </thead>
    <tbody>
      %s
    </tbody>
  </table>

  <div class="footer">
    <div>Engine: <strong>AuditForge v1.0</strong> (Deterministic Merkle DAG & WAL)</div>
    <div>Report ID: %s</div>
  </div>
</div>
</body>
</html>`,
		statusClass,
		statusText,
		verResult.VerifiedAt.Format(time.RFC3339),
		strings.ToUpper(manifest.Algorithm),
		manifest.TotalBlocks,
		manifest.StartBlockIndex,
		manifest.EndBlockIndex,
		manifest.TotalEvents,
		manifest.HeadBlockHash,
		manifest.SignerKeyID,
		rows.String(),
		truncate(manifest.HeadBlockHash, 12),
	)

	return html
}

func truncate(s string, l int) string {
	if len(s) <= l {
		return s
	}
	return s[:l]
}
