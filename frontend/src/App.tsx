import { useState, useEffect, useRef, useCallback } from 'react';
import './App.css';
import { GetStats, GetLogs, IsProtectionEnabled, ToggleProtection, TriggerUpdate, GetDatabaseInfo, GetConfig, SaveConfig } from "../wailsjs/go/main/App";
import { config, filter, logger, updater } from "../wailsjs/go/models";
type Stats = filter.Stats;
type LogEntry = logger.LogEntry;
type Source = updater.Source;
type Tab = 'dashboard' | 'sources';

// ─── Types ───────────────────────────────────────────────

interface LogViewProps {
  logs: LogEntry[];
  onClear: () => void;
}

interface StatCardProps {
  label: string;
  value: string | number;
  unit?: string;
  color: string;
}

interface AddSourceForm {
  name: string;
  url: string;
  format: number;
  apiKey: string;
  description: string;
}

const FORMAT_LABELS: Record<number, string> = {
  1: 'P2P Text',
  2: 'DAT',
  3: 'CIDR',
  4: 'Zakres IP',
};

const FORMAT_OPTIONS = [
  { value: 3, label: 'CIDR (1.2.3.0/24)' },
  { value: 4, label: 'Zakres (1.2.3.0-1.2.3.255)' },
  { value: 1, label: 'P2P Text (Level1:1.2.3.0-1.2.3.255)' },
  { value: 2, label: 'DAT (1.2.3.0 - 1.2.3.255 , 100 , Name)' },
];

// ─── Stat Card ───────────────────────────────────────────

function StatCard({ label, value, unit, color }: StatCardProps) {
  return (
    <div className="stat-card" style={{ borderLeft: `4px solid ${color}` }}>
      <div className="stat-label">{label}</div>
      <div className="stat-value">
        {typeof value === 'number' ? value.toLocaleString() : value}
        {unit && <span className="stat-unit">{unit}</span>}
      </div>
    </div>
  );
}

// ─── Log View ────────────────────────────────────────────

function LogView({ logs, onClear }: LogViewProps) {
  const logEndRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [filter_, setFilter_] = useState<string>('ALL');

  const filteredLogs = logs.filter(e => {
    if (filter_ === 'ALL') return true;
    if (filter_ === 'BLOCKED') return e.message.includes('BLOCK');
    if (filter_ === 'ERROR') return e.level >= 3;
    return true;
  });

  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [filteredLogs.length, autoScroll]);

  const getLevelClass = (level: number) => {
    switch (level) {
      case 0: return 'log-debug';
      case 1: return 'log-info';
      case 2: return 'log-warn';
      case 3: return 'log-error';
      default: return '';
    }
  };

  const getLevelLabel = (level: number) => {
    switch (level) {
      case 0: return 'DBG';
      case 1: return 'INF';
      case 2: return 'WRN';
      case 3: return 'ERR';
      default: return '?';
    }
  };

  return (
    <div className="log-view">
      <div className="log-toolbar">
        <span className="log-title">Logi zdarzeń</span>
        <select value={filter_} onChange={e => setFilter_(e.target.value)} className="log-filter">
          <option value="ALL">Wszystkie</option>
          <option value="BLOCKED">Blokady</option>
          <option value="ERROR">Błędy</option>
        </select>
        <label className="log-autoscroll">
          <input type="checkbox" checked={autoScroll} onChange={e => setAutoScroll(e.target.checked)} />
          Auto-scroll
        </label>
        <button className="log-clear-btn" onClick={onClear}>Wyczyść</button>
      </div>
      <div className="log-entries">
        {filteredLogs.length === 0 ? (
          <div className="log-empty">Brak zdarzeń...</div>
        ) : (
          filteredLogs.map((e, i) => (
            <div key={i} className={`log-entry ${getLevelClass(e.level)}`}>
              <span className="log-level">{getLevelLabel(e.level)}</span>
              <span className="log-msg">{e.message}</span>
            </div>
          ))
        )}
        <div ref={logEndRef} />
      </div>
    </div>
  );
}

// ─── Add Source Dialog ───────────────────────────────────

function AddSourceDialog({ onClose, onSave }: {
  onClose: () => void;
  onSave: (src: AddSourceForm) => void;
}) {
  const [form, setForm] = useState<AddSourceForm>({
    name: '', url: '', format: 3, apiKey: '', description: '',
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim() || !form.url.trim()) return;
    onSave(form);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h3>Dodaj źródło</h3>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <label className="form-field">
              <span>Nazwa</span>
              <input
                type="text" value={form.name}
                onChange={e => setForm({ ...form, name: e.target.value })}
                placeholder="np. moja-lista"
                required
              />
            </label>
            <label className="form-field">
              <span>URL</span>
              <input
                type="url" value={form.url}
                onChange={e => setForm({ ...form, url: e.target.value })}
                placeholder="https://example.com/blocklist.txt"
                required
              />
            </label>
            <label className="form-field">
              <span>Format</span>
              <select value={form.format} onChange={e => setForm({ ...form, format: Number(e.target.value) })}>
                {FORMAT_OPTIONS.map(o => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </label>
            <label className="form-field">
              <span>API Key (opcjonalne)</span>
              <input
                type="text" value={form.apiKey}
                onChange={e => setForm({ ...form, apiKey: e.target.value })}
                placeholder="Zostaw puste jeśli nie wymagane"
              />
            </label>
            <label className="form-field">
              <span>Opis (opcjonalne)</span>
              <textarea
                value={form.description}
                onChange={e => setForm({ ...form, description: e.target.value })}
                placeholder="Co ta lista blokuje?"
                rows={2}
              />
            </label>
          </div>
          <div className="modal-footer">
            <button type="button" className="btn-secondary" onClick={onClose}>Anuluj</button>
            <button type="submit" className="btn-primary">Dodaj źródło</button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Sources View ────────────────────────────────────────

function SourcesView({ onUpdate }: { onUpdate: () => void }) {
  const [cfg, setCfg] = useState<config.Config | null>(null);
  const [saving, setSaving] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [deletingIndex, setDeletingIndex] = useState<number | null>(null);

  const loadConfig = useCallback(async () => {
    try {
      const c = await GetConfig();
      setCfg(c);
    } catch (err) {
      console.error('load config error', err);
    }
  }, []);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const saveSources = async (sources: Source[]) => {
    const updated = new config.Config({ ...cfg, sources });
    setCfg(updated);
    setSaving(true);
    try {
      await SaveConfig(updated);
    } catch (err) {
      console.error('save config error', err);
    }
    setSaving(false);
  };

  const toggleSource = async (index: number) => {
    if (!cfg) return;
    const sources = cfg.sources.map((s, i) =>
      i === index ? new updater.Source({ ...s, enabled: !s.enabled }) : s
    );
    await saveSources(sources);
  };

  const handleAdd = async (form: AddSourceForm) => {
    if (!cfg) return;
    const sources = [...cfg.sources, new updater.Source({
      name: form.name,
      url: form.url,
      format: form.format,
      enabled: true,
      api_key: form.apiKey || '',
      description: form.description || '',
    })];
    await saveSources(sources);
    setShowAdd(false);
  };

  const handleDelete = async (index: number) => {
    if (!cfg) return;
    const sources = cfg.sources.filter((_, i) => i !== index);
    await saveSources(sources);
    setDeletingIndex(null);
  };

  if (!cfg) {
    return <div className="sources-loading">Ładowanie konfiguracji...</div>;
  }

  return (
    <div className="sources-view">
      <div className="sources-header">
        <h2>Źródła list IP ({cfg.sources.length})</h2>
        <div className="sources-actions">
          <button className="btn-secondary" onClick={() => setShowAdd(true)}>
            + Dodaj źródło
          </button>
          <button className="update-btn" onClick={onUpdate}>
            ↻ Aktualizuj teraz
          </button>
        </div>
      </div>
      <p className="sources-desc">
        Włącz lub wyłącz źródła blokad IP. Możesz dodać własne źródła z opcjonalnym kluczem API.
        Kliknij "Aktualizuj teraz" aby pobrać wybrane listy.
      </p>
      <div className="sources-list">
        {cfg.sources.map((src, i) => (
          <div key={i} className={`source-card ${src.enabled ? 'enabled' : 'disabled'}`}>
            <div className="source-main">
              <div className="source-info">
                <div className="source-name">{src.name}</div>
                {src.description && (
                  <div className="source-desc">{src.description}</div>
                )}
                <div className="source-url" title={src.url}>{src.url}</div>
                <div className="source-meta">
                  <span className="source-format-badge">{FORMAT_LABELS[src.format] || 'CIDR'}</span>
                  {src.api_key && (
                    <span className="source-api-badge" title={`API Key: ${src.api_key.substring(0, 8)}...`}>
                      🔑 API
                    </span>
                  )}
                  {src.last_sync && (
                    <span className="source-last-sync">
                      Ostatnia: {new Date(src.last_sync).toLocaleString('pl-PL')}
                    </span>
                  )}
                </div>
              </div>
              <div className="source-controls">
                <label className="source-toggle" onClick={e => e.stopPropagation()}>
                  <input
                    type="checkbox"
                    checked={src.enabled}
                    onChange={() => toggleSource(i)}
                  />
                  <span className="toggle-track">
                    <span className="toggle-indicator" />
                  </span>
                  <span className="toggle-status">{src.enabled ? 'Aktywne' : 'Wył.'}</span>
                </label>
                <button
                  className="source-delete-btn"
                  onClick={() => setDeletingIndex(i)}
                  title="Usuń źródło"
                >
                  🗑
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
      {saving && <div className="sources-saving">Zapisywanie...</div>}

      {/* Delete confirmation */}
      {deletingIndex !== null && (
        <div className="modal-overlay" onClick={() => setDeletingIndex(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <p className="modal-confirm-text">
              Usunąć źródło <strong>{cfg.sources[deletingIndex]?.name}</strong>?
            </p>
            <div className="modal-footer">
              <button className="btn-secondary" onClick={() => setDeletingIndex(null)}>Anuluj</button>
              <button className="btn-danger" onClick={() => handleDelete(deletingIndex)}>Usuń</button>
            </div>
          </div>
        </div>
      )}

      {/* Add source dialog */}
      {showAdd && (
        <AddSourceDialog
          onClose={() => setShowAdd(false)}
          onSave={handleAdd}
        />
      )}
    </div>
  );
}

// ─── Dashboard ───────────────────────────────────────────

function Dashboard({ stats, uptime, dbInfo, protected_, onToggle }: {
  stats: Stats | null;
  uptime: string;
  dbInfo: Record<string, any>;
  protected_: boolean;
  onToggle: () => void;
}) {
  const blockedRate = stats && (stats.blocked + stats.allowed) > 0
    ? ((stats.blocked / (stats.blocked + stats.allowed)) * 100).toFixed(1)
    : '0.0';

  return (
    <>
      <div className="toggle-section">
        <button
          className={`toggle-btn ${protected_ ? 'on' : 'off'}`}
          onClick={onToggle}
        >
          <div className="toggle-knob" />
          <span className="toggle-label">
            {protected_ ? 'Ochrona aktywna' : 'Ochrona wyłączona'}
          </span>
        </button>
      </div>

      <div className="stats-grid">
        <StatCard label="Zablokowane" value={stats?.blocked ?? 0} color="#ef4444" />
        <StatCard label="Przepuszczone" value={stats?.allowed ?? 0} color="#22c55e" />
        <StatCard label="Współczynnik blokad" value={blockedRate} unit="%" color="#f59e0b" />
        <StatCard label="Zakresy IP" value={dbInfo['ranges'] ?? 0} color="#3b82f6" />
        <StatCard label="Uptime" value={uptime} color="#8b5cf6" />
        <StatCard label="Upuszczone" value={stats?.dropped ?? 0} color="#64748b" />
      </div>
    </>
  );
}

// ─── Main App ────────────────────────────────────────────

function App() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [protected_, setProtected_] = useState(false);
  const [dbInfo, setDbInfo] = useState<Record<string, any>>({});
  const [uptime, setUptime] = useState('0s');
  const [tab, setTab] = useState<Tab>('dashboard');

  const refresh = useCallback(async () => {
    try {
      const s = await GetStats();
      setStats(s);
      const p = await IsProtectionEnabled();
      setProtected_(p);
      const d = await GetDatabaseInfo();
      setDbInfo(d);
      const l = await GetLogs(200);
      setLogs(l);
    } catch (err) {
      console.error('refresh error', err);
    }
  }, []);

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, 2000);
    return () => clearInterval(interval);
  }, [refresh]);

  useEffect(() => {
    if (!stats?.started_at || stats.started_at === 0) return;
    const start = Math.floor(stats.started_at / 1_000_000);
    const update = () => {
      const diff = Math.floor((Date.now() - start) / 1000);
      const h = Math.floor(diff / 3600);
      const m = Math.floor((diff % 3600) / 60);
      const s = diff % 60;
      setUptime(h > 0 ? `${h}h ${m}m` : m > 0 ? `${m}m ${s}s` : `${s}s`);
    };
    update();
    const iv = setInterval(update, 1000);
    return () => clearInterval(iv);
  }, [stats?.started_at]);

  const handleToggle = async () => {
    await ToggleProtection();
    setProtected_(!protected_);
  };

  const handleUpdate = async () => {
    await TriggerUpdate();
  };

  const handleClearLogs = () => {
    setLogs([]);
  };

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-left">
          <h1 className="app-title">go-peerblock</h1>
          <span className="app-subtitle">IP Blocker dla Windows</span>
        </div>
        <div className="header-right">
          <button className="update-btn" onClick={handleUpdate} title="Aktualizuj listy IP">
            ↻ Aktualizuj
          </button>
        </div>
      </header>

      <nav className="tab-nav">
        <button
          className={`tab-btn ${tab === 'dashboard' ? 'active' : ''}`}
          onClick={() => setTab('dashboard')}
        >📊 Dashboard</button>
        <button
          className={`tab-btn ${tab === 'sources' ? 'active' : ''}`}
          onClick={() => setTab('sources')}
        >📋 Źródła list IP</button>
      </nav>

      <main className="app-main">
        {tab === 'dashboard' && (
          <Dashboard
            stats={stats} uptime={uptime} dbInfo={dbInfo}
            protected_={protected_} onToggle={handleToggle}
          />
        )}
        {tab === 'sources' && <SourcesView onUpdate={handleUpdate} />}
        <LogView logs={logs} onClear={handleClearLogs} />
      </main>

      <footer className="status-bar">
        <span className={`status-dot ${protected_ ? 'active' : 'inactive'}`} />
        <span>{protected_ ? 'Ochrona aktywna' : 'Ochrona wyłączona'}</span>
        {stats && (
          <span className="status-fps">
            Pakiety: {(stats.blocked + stats.allowed).toLocaleString()}
          </span>
        )}
        <span className="status-tab">{tab === 'dashboard' ? 'Dashboard' : 'Źródła'}</span>
      </footer>
    </div>
  );
}

export default App;
