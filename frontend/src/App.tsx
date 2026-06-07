import { useState, useEffect, useCallback } from 'react';
import './App.css';
import appIcon from './assets/ikona.png';
import { GetStats, GetLogs, IsProtectionEnabled, ToggleProtection, TriggerUpdate, GetDatabaseInfo, GetCacheInfo, MinimizeToTray } from "../wailsjs/go/main/App";
import { filter, logger } from "../wailsjs/go/models";
import { Dashboard } from './components/Dashboard';
import { SourcesView } from './components/SourcesView';
import { SettingsView } from './components/SettingsView';
import { LogView } from './components/LogView';

type Stats = filter.Stats;
type LogEntry = logger.LogEntry;
type Tab = 'dashboard' | 'sources' | 'settings';

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
          <img className="app-logo" src={appIcon} alt="GO PeerBlock" />
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
