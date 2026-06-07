import { useState, useEffect, useCallback, useRef } from 'react';
import './App.css';
import appIcon from './assets/ikona.png';
import { GetStats, GetLogs, IsProtectionEnabled, ToggleProtection, TriggerUpdate, GetDatabaseInfo, GetCacheInfo, MinimizeToTray } from "../wailsjs/go/main/App";
import { filter, logger } from "../wailsjs/go/models";
import { Dashboard } from './components/Dashboard';
import { SourcesView } from './components/SourcesView';
import { SettingsView } from './components/SettingsView';
import { LogView } from './components/LogView';
import { ChartsView, type Sample } from './components/ChartsView';

type Stats = filter.Stats;
type LogEntry = logger.LogEntry;
type Tab = 'dashboard' | 'sources' | 'settings' | 'charts';

// ─── Main App ────────────────────────────────────────────

function App() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [protected_, setProtected_] = useState(false);
  const [dbInfo, setDbInfo] = useState<Record<string, any>>({});
  const [uptime, setUptime] = useState('0s');
  const [tab, setTab] = useState<Tab>('dashboard');
  const [cacheInfo, setCacheInfo] = useState<Record<string, any>>({});
  const [history, setHistory] = useState<Sample[]>([]);
  const prevStatsRef = useRef<Stats | null>(null);
  const prevTimeRef = useRef<number>(0);
  const collectingRef = useRef(false);

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

      // Calculate PPS delta from previous snapshot for the chart (only when tab active)
      if (collectingRef.current && prevStatsRef.current) {
        const now = Date.now();
        const elapsed = (now - prevTimeRef.current) / 1000;
        if (elapsed >= 0.5) {
          const blockedDelta = s.blocked - prevStatsRef.current.blocked;
          const allowedDelta = s.allowed - prevStatsRef.current.allowed;
          if (blockedDelta >= 0 && allowedDelta >= 0) {
            setHistory(prev => {
              const next = [...prev, {
                time: now,
                blockedPPS: blockedDelta / elapsed,
                allowedPPS: allowedDelta / elapsed,
              }];
              // Keep max 30 minutes of samples
              const cutoff = now - 30 * 60 * 1000;
              return next.filter(h => h.time >= cutoff);
            });
          }
        }
      }
      prevStatsRef.current = s;
      prevTimeRef.current = Date.now();
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

  // Pause chart data collection when the tab isn't active
  useEffect(() => {
    collectingRef.current = tab === 'charts';
  }, [tab]);

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
          className={`tab-btn ${tab === 'charts' ? 'active' : ''}`}
          onClick={() => setTab('charts')}
        >📈 Wykresy</button>
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
        {tab === 'charts' && <ChartsView history={history} />}
        <LogView logs={logs} onClear={handleClearLogs} />
      </main>

      <footer className="status-bar">
        <span className={`status-dot ${protected_ ? 'active' : 'inactive'}`} />
        <span>{protected_ ? 'Ochrona aktywna' : 'Ochrona wyłączona'}</span>
        {stats && (() => {
          const total = stats.blocked + stats.allowed;
          const elapsed = stats.started_at > 0
            ? (Date.now() - Math.floor(stats.started_at / 1_000_000)) / 1000
            : 0;
          const pps = elapsed > 0 ? (total / elapsed).toFixed(1) : '0.0';
          return (
            <span className="status-fps">
              Pakiety: {total.toLocaleString()} ({pps}/s)
            </span>
          );
        })()}
        <span className="status-tab">{
          tab === 'dashboard' ? 'Dashboard' : tab === 'sources' ? 'Źródła' : tab === 'charts' ? 'Wykresy' : 'Ustawienia'
        }</span>
      </footer>
    </div>
  );
}

export default App;
