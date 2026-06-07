import { useState, useEffect, useRef, useCallback } from 'react';
import './App.css';
import { GetStats, GetLogs, IsProtectionEnabled, ToggleProtection, TriggerUpdate, GetDatabaseInfo, GetConfig, SaveConfig, GetCacheInfo, MinimizeToTray } from "../wailsjs/go/main/App";
import { config, filter, logger, updater } from "../wailsjs/go/models";
type Stats = filter.Stats;
type LogEntry = logger.LogEntry;
type Source = updater.Source;
type Tab = 'dashboard' | 'sources' | 'settings';

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

  const getLevelClass = (level: number, msg: string) => {
    if (msg.startsWith('BLOCK')) return 'log-blocked';
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
            <div key={i} className={`log-entry ${getLevelClass(e.level, e.message)}`}>
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

function SourcesView({ onUpdate, updating }: { onUpdate: () => void; updating: boolean }) {
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
          <button className="update-btn" onClick={onUpdate} disabled={updating}>
            {updating ? '⏳' : '↻'} Aktualizuj
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

// ─── Settings View ─────────────────────────────────────

function SettingsView() {
  const [cfg, setCfg] = useState<config.Config | null>(null);
  const [allowlistText, setAllowlistText] = useState('');
  const [workerCount, setWorkerCount] = useState('0');
  const [cacheSize, setCacheSize] = useState('65536');
  const [cacheTtl, setCacheTtl] = useState('5');
  const [updateInterval, setUpdateInterval] = useState('24');
  const [logLevel, setLogLevel] = useState('info');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  const loadConfig = useCallback(async () => {
    try {
      const c = await GetConfig();
      setCfg(c);
      setAllowlistText((c.allowlist || []).join('\n'));
      setWorkerCount(String(c.worker_count ?? 0));
      setCacheSize(String(c.cache_size ?? 65536));
      // time.Duration is nanoseconds in JSON, convert to minutes
      const ttlNs = c.cache_ttl ?? 300000000000;
      setCacheTtl(String(Math.round(ttlNs / 60000000000)));
      const intervalNs = c.update_interval ?? 86400000000000;
      setUpdateInterval(String(Math.round(intervalNs / 3600000000000)));
      setLogLevel(c.log_level || 'info');
    } catch (err) {
      console.error('load config error', err);
    }
  }, []);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const handleSave = async () => {
    if (!cfg) return;
    setSaving(true);
    setSaved(false);
    setError('');
    try {
      const allowlist = allowlistText
        .split('\n')
        .map(l => l.trim())
        .filter(l => l !== '' && !l.startsWith('#'));
      
      const wc = parseInt(workerCount, 10);
      if (isNaN(wc) || wc < 0) throw new Error('Nieprawidłowa liczba workerów');
      const cs = parseInt(cacheSize, 10);
      if (isNaN(cs) || cs < 1) throw new Error('Nieprawidłowy rozmiar cache');
      const ttl = parseInt(cacheTtl, 10);
      if (isNaN(ttl) || ttl < 1) throw new Error('Nieprawidłowy TTL');
      const interval = parseInt(updateInterval, 10);
      if (isNaN(interval) || interval < 1) throw new Error('Nieprawidłowy interwał');

      const updated = new config.Config({
        ...cfg,
        allowlist,
        worker_count: wc,
        cache_size: cs,
        cache_ttl: ttl * 60000000000, // minutes → nanoseconds
        update_interval: interval * 3600000000000, // hours → nanoseconds
        log_level: logLevel,
      });
      await SaveConfig(updated);
      setCfg(updated);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err: any) {
      setError(err.message || 'Błąd zapisu');
      setTimeout(() => setError(''), 5000);
    }
    setSaving(false);
  };

  if (!cfg) {
    return <div className="sources-loading">Ładowanie ustawień...</div>;
  }

  return (
    <div className="settings-view">
      <div className="settings-section">
        <h3>Allowlista</h3>
        <div className="form-field">
          <span>Adresy IP, CIDR lub domeny (jeden na linię)</span>
          <textarea
            className="settings-textarea"
            value={allowlistText}
            onChange={e => setAllowlistText(e.target.value)}
            placeholder="8.8.8.8&#10;192.168.0.0/16&#10;*.example.com"
            rows={6}
          />
        </div>
      </div>

      <div className="settings-section">
        <h3>Wydajność</h3>
        <div className="settings-row">
          <span className="settings-label">Liczba workerów <code>worker_count</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={workerCount}
              onChange={e => setWorkerCount(e.target.value)}
              min="0"
              max="64"
            />
            <div className="settings-description">0 = automatycznie (liczba CPU)</div>
          </div>
        </div>
        <div className="settings-row">
          <span className="settings-label">Rozmiar cache <code>cache_size</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={cacheSize}
              onChange={e => setCacheSize(e.target.value)}
              min="1024"
              max="1048576"
            />
            <div className="settings-description">Liczba wpisów w cache decyzji</div>
          </div>
        </div>
        <div className="settings-row">
          <span className="settings-label">Cache TTL <code>cache_ttl</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={cacheTtl}
              onChange={e => setCacheTtl(e.target.value)}
              min="1"
              max="1440"
            />
            <span style={{ color: 'var(--text-muted)', fontSize: 12, marginLeft: 8 }}>minut</span>
          </div>
        </div>
      </div>

      <div className="settings-section">
        <h3>Aktualizacje</h3>
        <div className="settings-row">
          <span className="settings-label">Interwał aktualizacji <code>update_interval</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={updateInterval}
              onChange={e => setUpdateInterval(e.target.value)}
              min="1"
              max="168"
            />
            <span style={{ color: 'var(--text-muted)', fontSize: 12, marginLeft: 8 }}>godzin</span>
          </div>
        </div>
      </div>

      <div className="settings-section">
        <h3>Logowanie</h3>
        <div className="settings-row">
          <span className="settings-label">Poziom logowania <code>log_level</code></span>
          <select
            className="settings-select"
            value={logLevel}
            onChange={e => setLogLevel(e.target.value)}
          >
            <option value="debug">DEBUG — wszystko</option>
            <option value="info">INFO — informacje + błędy</option>
            <option value="warn">WARN — tylko ostrzeżenia + błędy</option>
            <option value="error">ERROR — tylko błędy</option>
          </select>
        </div>
      </div>

      <div className="settings-actions">
        <button className="btn-primary" onClick={handleSave} disabled={saving}>
          {saving ? '⏳ Zapisywanie...' : '💾 Zapisz ustawienia'}
        </button>
        {saved && <span className="settings-saved">✅ Zapisano!</span>}
        {error && <span className="settings-error">{error}</span>}
      </div>
    </div>
  );
}

// ─── Dashboard ───────────────────────────────────────────

function Dashboard({ stats, uptime, dbInfo, cacheInfo, protected_, onToggle }: {
  stats: Stats | null;
  uptime: string;
  dbInfo: Record<string, any>;
  cacheInfo: Record<string, any>;
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
        <StatCard
          label="Cache"
          value={`${(cacheInfo['entries'] ?? 0).toLocaleString()} / ${(cacheInfo['max'] ?? 65536).toLocaleString()}`}
          color="#64748b"
        />
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
  const [cacheInfo, setCacheInfo] = useState<Record<string, any>>({});

  const refresh = useCallback(async () => {
    try {
      const s = await GetStats();
      setStats(s);
      const p = await IsProtectionEnabled();
      setProtected_(p);
      const d = await GetDatabaseInfo();
      setDbInfo(d);
      const ci = await GetCacheInfo();
      setCacheInfo(ci);
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

  const [updating, setUpdating] = useState(false);

  const handleUpdate = async () => {
    setUpdating(true);
    await TriggerUpdate();
    setTimeout(() => setUpdating(false), 3000);
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
          <button className="update-btn" onClick={handleUpdate} title="Aktualizuj listy IP" disabled={updating}>
            {updating ? '⏳' : '↻'} Aktualizuj
          </button>
          <button className="tray-btn" onClick={MinimizeToTray} title="Minimalizuj do zasobnika">
            ⬇
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
        <button
          className={`tab-btn ${tab === 'settings' ? 'active' : ''}`}
          onClick={() => setTab('settings')}
        >⚙️ Ustawienia</button>
      </nav>

      <main className="app-main">
        {tab === 'dashboard' && (
          <Dashboard
            stats={stats} uptime={uptime} dbInfo={dbInfo} cacheInfo={cacheInfo}
            protected_={protected_} onToggle={handleToggle}
          />
        )}
        {tab === 'sources' && <SourcesView onUpdate={handleUpdate} updating={updating} />}
        {tab === 'settings' && <SettingsView />}
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
        <span className="status-tab">{
          tab === 'dashboard' ? 'Dashboard' : tab === 'sources' ? 'Źródła' : 'Ustawienia'
        }</span>
      </footer>
    </div>
  );
}

export default App;
