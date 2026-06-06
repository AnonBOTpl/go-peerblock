import { useState, useEffect, useRef, useCallback, type FC } from 'react';
import './App.css';
import { GetStats, GetLogs, IsProtectionEnabled, ToggleProtection, TriggerUpdate, GetDatabaseInfo } from "../wailsjs/go/main/App";
import { filter, logger } from "../wailsjs/go/models";
type Stats = filter.Stats;
type LogEntry = logger.LogEntry;

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
  const [filter, setFilter] = useState<string>('ALL');

  const filteredLogs = logs.filter(e => {
    if (filter === 'ALL') return true;
    if (filter === 'BLOCKED') return e.level === 0 && e.message.includes('BLOCK');
    if (filter === 'ERROR') return e.level >= 3;
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
        <select value={filter} onChange={e => setFilter(e.target.value)} className="log-filter">
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

// ─── Main App ────────────────────────────────────────────

function App() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [protected_, setProtected_] = useState(false);
  const [dbInfo, setDbInfo] = useState<Record<string, any>>({});
  const [uptime, setUptime] = useState('0s');

  // Fetch stats periodically
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

  // Format uptime (started_at is UnixNano from Go)
  useEffect(() => {
    if (!stats?.started_at || stats.started_at === 0) return;
    const start = Math.floor(stats.started_at / 1_000_000); // UnixNano → ms
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

  const blockedRate = stats && (stats.blocked + stats.allowed) > 0
    ? ((stats.blocked / (stats.blocked + stats.allowed)) * 100).toFixed(1)
    : '0.0';

  return (
    <div className="app">
      {/* Header */}
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

      {/* Main Content */}
      <main className="app-main">
        {/* Protection Toggle */}
        <div className="toggle-section">
          <button
            className={`toggle-btn ${protected_ ? 'on' : 'off'}`}
            onClick={handleToggle}
          >
            <div className="toggle-knob" />
            <span className="toggle-label">
              {protected_ ? 'Ochrona aktywna' : 'Ochrona wyłączona'}
            </span>
          </button>
        </div>

        {/* Stats Cards */}
        <div className="stats-grid">
          <StatCard
            label="Zablokowane"
            value={stats?.blocked ?? 0}
            color="#ef4444"
          />
          <StatCard
            label="Przepuszczone"
            value={stats?.allowed ?? 0}
            color="#22c55e"
          />
          <StatCard
            label="Współczynnik blokad"
            value={blockedRate}
            unit="%"
            color="#f59e0b"
          />
          <StatCard
            label="Zakresy IP"
            value={dbInfo['ranges'] ?? 0}
            color="#3b82f6"
          />
          <StatCard
            label="Uptime"
            value={uptime}
            color="#8b5cf6"
          />
          <StatCard
            label="Upuszczone"
            value={stats?.dropped ?? 0}
            color="#64748b"
          />
        </div>

        {/* Logs */}
        <LogView logs={logs} onClear={handleClearLogs} />
      </main>

      {/* Status Bar */}
      <footer className="status-bar">
        <span className={`status-dot ${protected_ ? 'active' : 'inactive'}`} />
        <span>{protected_ ? 'Ochrona aktywna' : 'Ochrona wyłączona'}</span>
        {stats && (
          <span className="status-fps">
            Pakiety: {stats.blocked + stats.allowed}
          </span>
        )}
      </footer>
    </div>
  );
}

export default App;
